package telemetry

import (
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporters/otlp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func Init() error {
	// Instantiate the exporter.
	exporter, err := otlp.NewExporter(
		otlp.WithAddress("agent:55680"),
		otlp.WithInsecure(),
	)
	if err != nil {
		return err
	}

	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithBatcher(exporter),
	)
	if err != nil {
		return err
	}

	global.SetTraceProvider(tp)

	return nil
}
