package node

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelLog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkLog "go.opentelemetry.io/otel/sdk/log"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/alcionai/clues/internal/stringify"
)

// ------------------------------------------------------------
// client
// ------------------------------------------------------------

type OTELClient struct {
	ServiceName string
	grpcConn    *grpc.ClientConn

	LoggerProvider *sdkLog.LoggerProvider
	Logger         otelLog.Logger

	MeterProvider *sdkMetric.MeterProvider
	Meter         metric.Meter

	TracerProvider *sdkTrace.TracerProvider
	Tracer         trace.Tracer
}

func (cli *OTELClient) Close(ctx context.Context) error {
	if cli == nil {
		return nil
	}

	if cli.MeterProvider != nil {
		err := cli.MeterProvider.ForceFlush(ctx)
		if err != nil {
			log.Println("forcing meter provider flush:", err)
		}

		err = cli.MeterProvider.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("shutting down otel meterprovider: %w", err)
		}
	}

	if cli.LoggerProvider != nil {
		err := cli.LoggerProvider.ForceFlush(ctx)
		if err != nil {
			log.Println("forcing trace provider flush:", err)
		}

		err = cli.LoggerProvider.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("shutting down otel logger provider: %w", err)
		}
	}

	if cli.TracerProvider != nil {
		err := cli.TracerProvider.ForceFlush(ctx)
		if err != nil {
			log.Println("forcing trace provider flush:", err)
		}

		err = cli.TracerProvider.Shutdown(ctx)
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
	// Resource contains information about the thing sourcing logs, metrics, and
	// traces in OTEL. This information will be available in backends on all logs
	// traces, and metrics that are generated from this source.
	//
	// The provided resource should represent the service that's initializing
	// clues. The resource should encapsulate all parts of the metrics that need
	// reporting, not just a subset of them (i.e. it represents the "root" of the
	// information that will be reported to OTEL).
	//
	// If not provided, a minimal Resource containing the service name will be
	// created.
	Resource *resource.Resource

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
	var (
		err    error
		server = config.Resource
	)

	if server == nil {
		server, err = resource.New(ctx, resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName)))
		if err != nil {
			return nil, errors.Wrap(err, "creating otel server resource")
		}
	}

	// -- Client

	client := OTELClient{}

	// just a qol wrapper for shutting down on errors in this constructor.
	closeClient := func() {
		err := client.Close(ctx)
		if err != nil {
			log.Printf("err closing client: %v\n", err)
		}
	}

	// -- grpc client

	// Note the use of insecure transport here. TLS is recommended in production.
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())

	client.grpcConn, err = grpc.NewClient(config.GRPCEndpoint, creds)
	if err != nil {
		return nil, fmt.Errorf("creating new grpc connection: %w", err)
	}

	// -- Tracing

	client.TracerProvider, err = newTracerProvider(ctx, client.grpcConn, server)
	if err != nil {
		closeClient()
		return nil, errors.Wrap(err, "generating a tracer provider")
	}

	// set propagation to include traceContext and baggage (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{}))
	otel.SetTracerProvider(client.TracerProvider)
	client.Tracer = client.TracerProvider.Tracer(serviceName + "/tracer")

	// -- Logging

	// generate a logger provider
	// LoggerProvider := global.GetLoggerProvider()
	client.LoggerProvider, err = newLoggerProvider(ctx, client.grpcConn, server)
	if err != nil {
		closeClient()
		return nil, errors.Wrap(err, "generating a logger provider")
	}

	global.SetLoggerProvider(client.LoggerProvider)
	client.Logger = client.LoggerProvider.Logger(serviceName)

	// -- Metrics

	client.MeterProvider, err = newMeterProvider(ctx, client.grpcConn, server)
	if err != nil {
		closeClient()
		return nil, errors.Wrap(err, "generating a meter provider")
	}

	otel.SetMeterProvider(client.MeterProvider)
	client.Meter = client.MeterProvider.Meter(serviceName)

	// Shutdown will flush any remaining spans and shut down the exporter.
	return &client, nil
}

// newTracerProvider constructs a new tracer that manages batch exports
// of tracing values.
func newTracerProvider(
	ctx context.Context,
	conn *grpc.ClientConn,
	server *resource.Resource,
) (*sdkTrace.TracerProvider, error) {
	if ctx == nil {
		return nil, errors.New("nil ctx")
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, errors.Wrap(err, "constructing a tracer exporter")
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	batchSpanProcessor := sdkTrace.NewBatchSpanProcessor(exporter)

	tracerProvider := sdkTrace.NewTracerProvider(
		sdkTrace.WithResource(server),
		// FIXME: need to investigate other options...
		// * case handling for parent-not-sampled
		// * blocking on full queue
		// * max queue size
		// FIXME: need to refine trace sampling.
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

	return tracerProvider, nil
}

// newMeterProvider constructs a new meter that manages batch exports
// of metrics.
func newMeterProvider(
	ctx context.Context,
	conn *grpc.ClientConn,
	server *resource.Resource,
) (*sdkMetric.MeterProvider, error) {
	if ctx == nil {
		return nil, errors.New("nil ctx")
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithGRPCConn(conn),
		otlpmetricgrpc.WithCompressor("gzip"))
	if err != nil {
		return nil, errors.Wrap(err, "constructing a meter exporter")
	}

	periodicReader := sdkMetric.NewPeriodicReader(
		exporter,
		sdkMetric.WithInterval(1*time.Minute))

	meterProvider := sdkMetric.NewMeterProvider(
		sdkMetric.WithResource(server),
		// FIXME: need to investigate other options...
		// * view
		// * interval
		// * aggregation
		// * temporality
		sdkMetric.WithReader(periodicReader))

	return meterProvider, nil
}

// newLoggerProvider constructs a new logger that manages batch exports
// of logs.
func newLoggerProvider(
	ctx context.Context,
	conn *grpc.ClientConn,
	server *resource.Resource,
) (*sdkLog.LoggerProvider, error) {
	if ctx == nil {
		return nil, errors.New("nil ctx")
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, errors.Wrap(err, "constructing a logger exporter")
	}

	loggerProvider := sdkLog.NewLoggerProvider(
		sdkLog.WithResource(server),
		// FIXME: need to investigate other options...
		// * interval
		// * buffer size
		// * count limit
		// * value length limit
		sdkLog.WithProcessor(sdkLog.NewBatchProcessor(exporter)))

	return loggerProvider, nil
}

// ------------------------------------------------------------
// annotations.  basically otel's version of With()
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

func (a Annotation) KV() otelLog.KeyValue {
	if a.kind != "attribute" {
		return otelLog.KeyValue{}
	}

	// FIXME: needs extensive type support
	switch a.v.(type) {
	case int:
		return otelLog.Int(a.k, a.v.(int))
	case int64:
		return otelLog.Int64(a.k, a.v.(int64))
	case string:
		return otelLog.String(a.k, a.v.(string))
	case bool:
		return otelLog.Bool(a.k, a.v.(bool))
	default: // everything else gets stringified
		return otelLog.String(a.k, stringify.Marshal(a.v, false))
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

	ctx, span := dn.OTEL.Tracer.Start(ctx, name)

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

// logger gets the otel logger instance from the otel client.
// Returns nil if otel wasn't initialized.
func (dn *Node) OTELLogger() otelLog.Logger {
	if dn == nil || dn.OTEL == nil {
		return nil
	}

	return dn.OTEL.Logger
}

// OTELMeter gets the otel logger instance from the otel client.
// Returns nil if otel wasn't initialized.
func (dn *Node) OTELMeter() metric.Meter {
	if dn == nil || dn.OTEL == nil {
		return nil
	}

	return dn.OTEL.Meter
}
