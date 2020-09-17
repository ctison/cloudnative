package telemetry

import (
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporters/otlp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func Init(addr string) error {
	// Create the exporter.
	exporter, err := otlp.NewExporter(
		otlp.WithAddress(addr),
		otlp.WithInsecure(),
	)
	if err != nil {
		return err
	}

	// Create the trace provider.
	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithBatcher(exporter),
	)
	if err != nil {
		return err
	}

	// Set the trace provider as default.
	global.SetTraceProvider(tp)

	return nil
}
