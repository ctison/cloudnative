module github.com/ctison/cloudnative

go 1.15

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/golang/protobuf v1.4.2
	github.com/spf13/cobra v1.0.0
	go.elastic.co/apm/module/apmgin v1.8.0
	go.elastic.co/apm/module/apmzap v1.8.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin v0.11.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc v0.11.0
	go.opentelemetry.io/otel v0.11.0
	go.opentelemetry.io/otel/exporters/otlp v0.11.0
	go.opentelemetry.io/otel/sdk v0.11.0
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.32.0
	google.golang.org/protobuf v1.23.0
)
