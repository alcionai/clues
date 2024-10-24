package clues

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
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

type otelClient struct {
	serviceName     string
	exportsToStdOut bool

	// standard connections
	grpcConn      *grpc.ClientConn
	traceProvider *sdkTrace.TracerProvider
	tracer        trace.Tracer
	logger        otellog.Logger
}

func (cli *otelClient) close(
	ctx context.Context,
) error {
	if cli == nil {
		return nil
	}

	// FIXME: there's probably a better way to fix up temporal here.
	if ctx == nil {
		ctx = context.Background()
	}

	if cli.traceProvider != nil {
		err := cli.traceProvider.ForceFlush(ctx)
		if err != nil {
			log.Println("forcing trace provider flush:", err)
		}

		err = cli.traceProvider.Shutdown(ctx)
		if err != nil {
			return WrapWC(ctx, err, "shutting down otel trace provider")
		}
	}

	if cli.grpcConn != nil {
		err := cli.grpcConn.Close()
		if err != nil {
			return WrapWC(ctx, err, "closing grpc connection")
		}
	}

	return nil
}

func otelIsLive(cli *otelClient) bool {
	return cli != nil && cli.tracer != nil
}

// ------------------------------------------------------------
// initializers
// ------------------------------------------------------------

type OTELConfig struct {
	// specify the endpoint location to use for grpc communication.
	// If empty, no telemetry exporter will be generated.
	// ex: localhost:4317
	// ex: 0.0.0.0:4317
	GRPCEndpoint string

	// ExportToStdOut exports to stdout so that clients can
	// consume traces with the stdout feed instead of having
	// data relayed to a server.
	ExportToStdOut bool

	// AllowGlobalSpans tells clues that it may lose context of
	// the otel client across ctx serialization boundaries (this
	// happens in temporal).  If this is flagged, then clues will
	// attempt to eagerly check for spans in the context as well
	// as the data node.
	AllowGlobalSpans bool

	// UseTemporalTracer should be flagged anytime clues is working
	// within a temporal instance.
	UseTemporalTracer bool
}

// newOTELClient bootstraps the OpenTelemetry pipeline according to
// the provided configuration.  If UseGlobalTrace == false, otel will
// use a  local server instance; if true, it generates a global singleton.
// Global singletons may be required for frameworks that serialize
// context values, such as Temporal.
//
// If this does not return an error, make sure to call the
// client.Close() method for proper cleanup.
func newOTELClient(
	ctx context.Context,
	serviceName string,
	cfg OTELConfig,
) (*otelClient, error) {
	// -- Resource
	resource, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName)))
	if err != nil {
		return nil, Wrap(err, "creating otel resource")
	}

	// -- TracerProvider

	opts := []sdkTrace.TracerProviderOption{
		sdkTrace.WithSampler(sdkTrace.AlwaysSample()),
		sdkTrace.WithResource(resource),
		// FIXME: prod will need configuration for this
		sdkTrace.WithSampler(sdkTrace.AlwaysSample()),
		sdkTrace.WithRawSpanLimits(sdkTrace.SpanLimits{
			AttributeValueLengthLimit:   -1,
			AttributeCountLimit:         -1,
			AttributePerEventCountLimit: -1,
			AttributePerLinkCountLimit:  -1,
			EventCountLimit:             -1,
			LinkCountLimit:              -1,
		}),
	}

	// -- Exporter
	var grpcConn *grpc.ClientConn

	if cfg.ExportToStdOut {
		// Register a stdout trace exporter for cases that don't want
		// to export to a batch span processor.
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, Wrap(err, "generating stdOutTrace batcher")
		}

		opts = append(opts, sdkTrace.WithBatcher(exp))
	} else {
		// Register the trace exporter with a grps-bound batch
		// span processor to aggregate spans before export.
		grpcConn, err = grpc.NewClient(
			cfg.GRPCEndpoint,
			// Note the use of insecure transport here. TLS is recommended in production.
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, Wrap(err, "creating new gRPC connection")
		}

		exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(grpcConn))
		if err != nil {
			return nil, Wrap(err, "creating a grpc trace exporter")
		}

		batchSpanProcessor := sdkTrace.NewBatchSpanProcessor(exporter)

		opts = append(opts, sdkTrace.WithSpanProcessor(batchSpanProcessor))
	}

	tracerProvider := sdkTrace.NewTracerProvider(opts...)

	// set global propagator to traceContext (the default is no-op).
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{}))
	otel.SetTracerProvider(tracerProvider)

	// -- Logger

	// generate a logger provider
	logProvider := global.GetLoggerProvider()

	// -- Client

	client := otelClient{
		serviceName:     serviceName,
		exportsToStdOut: cfg.ExportToStdOut,
		grpcConn:        grpcConn,
		traceProvider:   tracerProvider,
		tracer:          tracerProvider.Tracer(serviceName + "/tracer"),
		logger:          logProvider.Logger(serviceName),
	}

	// Shutdown will flush any remaining spans and shut down the exporter.
	return &client, nil
}

// ------------------------------------------------------------
// annotations.  basically otel's version of With()
// ------------------------------------------------------------

type annotation struct {
	kind string
	k    string
	v    any
}

func NewAttribute(k string, v any) annotation {
	return annotation{
		kind: "attribute",
		k:    k,
		v:    v,
	}
}

func (a annotation) IsAttribute() bool {
	return a.kind == "attribute"
}

func (a annotation) KV() otellog.KeyValue {
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
		return otellog.String(a.k, marshal(a.v, false))
	}
}

type Annotationer interface {
	IsAttribute() bool
	KV() attribute.KeyValue
}
