module github.com/ctison/cloudnative

go 1.15

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/spf13/cobra v1.0.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin v0.11.0
	go.opentelemetry.io/otel v0.11.0
	go.opentelemetry.io/otel/exporters/otlp v0.11.0
	go.opentelemetry.io/otel/sdk v0.11.0
	go.uber.org/zap v1.16.0
)
