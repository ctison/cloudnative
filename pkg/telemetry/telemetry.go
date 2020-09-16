package telemetry

import (
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/sdk/trace"
)

func Init() error {
	exporter, err := otlp.NewExporter(
		otlp.WithInsecure(),
		otlp.WithAddress("agent:55680"),
	)
	if err != nil {
		return err
	}
	traceProvider, err := trace.NewProvider(
		trace.WithConfig(trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		}),
		trace.WithSyncer(exporter),
		// trace.WithResource(resource.New(label.String("service", "cloudnative"))),
	)
	if err != nil {
		return err
	}
	global.SetTraceProvider(traceProvider)
	return nil
}
