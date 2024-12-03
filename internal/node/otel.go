package node

import (
	"context"
	"fmt"
	"net/http"

	"github.com/alcionai/clues/internal/stringify"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ------------------------------------------------------------
// client
// ------------------------------------------------------------

type OTELClient struct {
	grpcConn      *grpc.ClientConn
	traceProvider *sdkTrace.TracerProvider
	tracer        trace.Tracer
	logger        otellog.Logger
	serviceName   string
}

func (cli *OTELClient) Close(ctx context.Context) error {
	if cli == nil {
		return nil
	}

	if cli.traceProvider != nil {
		err := cli.traceProvider.ForceFlush(ctx)
		if err != nil {
			fmt.Println("forcing trace provider flush:", err)
		}

		err = cli.traceProvider.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("shutting down otel trace provider: %w", err)
		}
	}

	if cli.grpcConn != nil {
		err := cli.grpcConn.Close()
		if err != nil {
			return fmt.Errorf("closing grpc connection: %w", err)
		}
	}

	return nil
}

// ------------------------------------------------------------
// config
// ------------------------------------------------------------

type OTELConfig struct {
	// specify the endpoint location to use for grpc communication.
	// If empty, no telemetry exporter will be generated.
	// ex: localhost:4317
	// ex: 0.0.0.0:4317
	GRPCEndpoint string
}

// ------------------------------------------------------------
// initializers
// ------------------------------------------------------------

// NewOTELClient bootstraps the OpenTelemetry pipeline to run against a
// local server instance. If it does not return an error, make sure
// to call the client.Close() method for proper cleanup.
// The service name is used to match traces across backends.
func NewOTELClient(
	ctx context.Context,
	serviceName string,
	config OTELConfig,
) (*OTELClient, error) {
	// -- Resource
	srvResource, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceNameKey.String(serviceName)))
	if err != nil {
		return nil, fmt.Errorf("creating otel resource: %w", err)
	}

	// -- Exporter

	conn, err := grpc.NewClient(
		config.GRPCEndpoint,
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("creating new grpc connection: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("creating a trace exporter: %w", err)
	}

	// -- TracerProvider

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	batchSpanProcessor := sdkTrace.NewBatchSpanProcessor(exporter)

	tracerProvider := sdkTrace.NewTracerProvider(
		sdkTrace.WithResource(srvResource),
		sdkTrace.WithSampler(sdkTrace.AlwaysSample()),
		sdkTrace.WithSpanProcessor(batchSpanProcessor),
		sdkTrace.WithRawSpanLimits(sdkTrace.SpanLimits{
			AttributeValueLengthLimit:   -1,
			AttributeCountLimit:         -1,
			AttributePerEventCountLimit: -1,
			AttributePerLinkCountLimit:  -1,
			EventCountLimit:             -1,
			LinkCountLimit:              -1,
		}))

	// set global propagator to traceContext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	// -- Logger

	// generate a logger provider
	logProvider := global.GetLoggerProvider()

	// -- Client

	client := OTELClient{
		grpcConn:      conn,
		traceProvider: tracerProvider,
		tracer:        tracerProvider.Tracer(serviceName + "/tracer"),
		logger:        logProvider.Logger(serviceName),
	}

	// Shutdown will flush any remaining spans and shut down the exporter.
	return &client, nil
}

// ------------------------------------------------------------
// annotations.  basically otel's version of With()
// Not currently used; we're just mashing everything in as a
// string right now, same as clues does.
// ------------------------------------------------------------

type Annotation struct {
	kind string
	k    string
	v    any
}

func NewAttribute(k string, v any) Annotation {
	return Annotation{
		kind: "attribute",
		k:    k,
		v:    v,
	}
}

func (a Annotation) IsAttribute() bool {
	return a.kind == "attribute"
}

func (a Annotation) KV() otellog.KeyValue {
	if a.kind != "attribute" {
		return otellog.KeyValue{}
	}

	// FIXME: needs extensive type support
	switch a.v.(type) {
	case int:
		return otellog.Int(a.k, a.v.(int))
	case int64:
		return otellog.Int64(a.k, a.v.(int64))
	case string:
		return otellog.String(a.k, a.v.(string))
	case bool:
		return otellog.Bool(a.k, a.v.(bool))
	default: // everything else gets stringified
		return otellog.String(a.k, stringify.Marshal(a.v, false))
	}
}

type Annotationer interface {
	IsAttribute() bool
	KV() attribute.KeyValue
}

// ------------------------------------------------------------
// span handling
// ------------------------------------------------------------

// TraceMapCarrierBase defines the structures that support
// otel TraceMapCarrier behavior.  A traceMapCarrier is used
// to pass and receive traces using message delivery headers
// and other metadata.
type TraceMapCarrierBase interface {
	map[string]string | http.Header
}

// AsTraceMapCarrier converts a traceMapCarrier interface to
// its propagation package implementation for that structure.
// ie: map becomes a MapCarrier, headers become HeaderCarriers.
func AsTraceMapCarrier[C TraceMapCarrierBase](
	carrier C,
) propagation.TextMapCarrier {
	if carrier == nil {
		return propagation.MapCarrier{}
	}

	if mss, ok := any(carrier).(map[string]string); ok {
		return propagation.MapCarrier(mss)
	}

	if hh, ok := any(carrier).(http.Header); ok {
		return propagation.HeaderCarrier(hh)
	}

	return propagation.MapCarrier{}
}

// injectTrace adds the current trace details to the provided
// carrier.  If otel is not initialized, no-ops.
//
// The carrier data is mutated by this call.
func (dn *Node) InjectTrace(
	ctx context.Context,
	carrier propagation.TextMapCarrier,
) {
	if dn == nil {
		return
	}

	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// receiveTrace extracts the current trace details from the
// carrier and adds them to the context.  If otel is not
// initialized, no-ops.
//
// The carrier data is mutated by this call.
func (dn *Node) ReceiveTrace(
	ctx context.Context,
	carrier propagation.TextMapCarrier,
) context.Context {
	if dn == nil {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// AddSpan adds a new otel span.  If the otel client is nil, no-ops.
// Attrs can be added to the span with addSpanAttrs.  This span will
// continue to be used for that purpose until replaced with another
// span, which will appear in a separate context (and thus a separate,
// node).
func (dn *Node) AddSpan(
	ctx context.Context,
	name string,
) (context.Context, *Node) {
	if dn == nil || dn.OTEL == nil {
		return ctx, dn
	}

	ctx, span := dn.OTEL.tracer.Start(ctx, name)

	spawn := dn.SpawnDescendant()
	spawn.Span = span

	return ctx, spawn
}

// CloseSpan closes the otel span and removes it span from the data node.
// If no span is present, no ops.
func (dn *Node) CloseSpan(ctx context.Context) *Node {
	if dn == nil || dn.Span == nil {
		return dn
	}

	dn.Span.End()

	spawn := dn.SpawnDescendant()
	spawn.Span = nil

	return spawn
}

// AddSpanAttributes adds the values to the current span.  If the span
// is nil (such as if otel wasn't initialized or no span has been generated),
// this call no-ops.
func (dn *Node) AddSpanAttributes(
	values map[string]any,
) {
	if dn == nil || dn.Span == nil {
		return
	}

	for k, v := range values {
		dn.Span.SetAttributes(attribute.String(k, stringify.Marshal(v, false)))
	}
}

// OTELLogger gets the otel logger instance from the otel client.
// Returns nil if otel wasn't initialized.
func (dn *Node) OTELLogger() otellog.Logger {
	if dn == nil || dn.OTEL == nil {
		return nil
	}

	return dn.OTEL.logger
}
