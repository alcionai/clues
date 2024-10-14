package clues

import (
	"context"
	"fmt"

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

type otelClient struct {
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

	// FIXME: there's probably a better way to fix up temporal
	// here.
	if ctx == nil {
		ctx = context.Background()
	}

	if cli.traceProvider != nil {
		err := cli.traceProvider.ForceFlush(ctx)
		if err != nil {
			fmt.Println("forcing trace provider flush:", err)
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

// ------------------------------------------------------------
// initializers
// ------------------------------------------------------------

type OTELConfig struct {
	// specify the endpoint location to use for grpc communication.
	// If empty, no telemetry exporter will be generated.
	// ex: localhost:4317
	// ex: 0.0.0.0:4317
	GRPCEndpoint string
}

// newOTELClient bootstraps the OpenTelemetry pipeline to run against a
// local server instance. If it does not return an error, make sure
// to call the client.Close() method for proper cleanup.
// The service name is used to match traces across backends.
func newOTELClient(
	ctx context.Context,
	serviceName string,
	config OTELConfig,
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

	// -- Exporter

	conn, err := grpc.NewClient(
		config.GRPCEndpoint,
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, Wrap(err, "creating new gRPC connection")
	}

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, Wrap(err, "creating a trace exporter")
	}

	// -- TracerProvider

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	batchSpanProcessor := sdkTrace.NewBatchSpanProcessor(exporter)

	tracerProvider := sdkTrace.NewTracerProvider(
		sdkTrace.WithResource(resource),
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

	client := otelClient{
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
