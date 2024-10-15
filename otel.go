package clues

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type otelClient struct {
	traceProvider *sdkTrace.TracerProvider
	tracer        trace.Tracer
	logger        otellog.Logger
}

// ------------------------------------------------------------
// initializers
// ------------------------------------------------------------

// newOTELClient bootstraps the OpenTelemetry pipeline to run against a
// local server instance. If it does not return an error, make sure
// to call the client.Close() method for proper cleanup.
func newOTELClient(ctx context.Context, serviceName string) (*otelClient, error) {
	// the service name is used to display traces in backends
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, WrapWC(ctx, err, "creating otel resource")
	}

	// If the OpenTelemetry Collector is running on a local cluster (minikube or
	// microk8s), it should be accessible through the NodePort service at the
	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
	// endpoint of your cluster. If you run the app inside k8s, then you can
	// probably connect directly to the service through dns.
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	conn, err := grpc.NewClient(serviceName,
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, WrapWC(ctx, err, "creating a gRPC connection to collector")
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, WrapWC(ctx, err, "creating a trace exporter")
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdkTrace.NewBatchSpanProcessor(traceExporter)

	tracerProvider := sdkTrace.NewTracerProvider(
		sdkTrace.WithSampler(sdkTrace.AlwaysSample()),
		sdkTrace.WithResource(res),
		sdkTrace.WithSpanProcessor(bsp))

	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	logProvider := global.GetLoggerProvider()

	client := otelClient{
		traceProvider: tracerProvider,
		tracer:        tracerProvider.Tracer(serviceName),
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
