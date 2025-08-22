package telemetry

import (
	"context"

	"github.com/abgdnv/gocommerce/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

func NewTracerProvider(ctx context.Context, serviceName string, cfg config.TelemetryConfig) (*tracesdk.TracerProvider, error) {

	collectorOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Traces.OtlpHttp.Endpoint),
		otlptracehttp.WithTimeout(cfg.Traces.OtlpHttp.Timeout),
	}
	if cfg.Traces.OtlpHttp.Insecure {
		collectorOpts = append(collectorOpts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, collectorOpts...)
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}
