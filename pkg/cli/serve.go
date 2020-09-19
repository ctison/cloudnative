package cli

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/ctison/cloudnative/pkg/server"
	"github.com/ctison/cloudnative/pkg/server/grpc"
	"github.com/ctison/cloudnative/pkg/server/http"
	"github.com/ctison/cloudnative/pkg/server/signal"
	"github.com/ctison/cloudnative/pkg/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.elastic.co/apm/module/apmzap"
	otel "go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.uber.org/zap"
)

var errExample = errors.New("example erro")

func (cli *CLI) Serve(cmd *cobra.Command, args []string) {
	// Instantiate logger.
	logConfig := zap.NewProductionConfig()
	logConfig.Level.SetLevel(cli.serve.logLevel.Level)
	logConfig.Development = cli.serve.devMode
	log, err := logConfig.Build(zap.WrapCore((&apmzap.Core{}).WrapCore))
	if err != nil {
		panic(err)
	}

	// Setup telemetry if enabled.
	if cli.serve.otelAddr != "" {
		if err := telemetry.Init(cli.serve.otelAddr); err != nil {
			log.Error("failed to start telemetry", zap.Error(err))
			os.Exit(1)
		}
	}
	tracer := otel.Tracer("cloudnative")

	// Setup HTTP server.
	httpServer := http.New(log, cli.serve.devMode, nil)
	r := httpServer.Gin()
	setupHTTP(r, tracer, log)

	// Setup gRPC server.
	grpcServer := grpc.New(log.Named("grpc"))

	// Instantiate all servers.
	srv := server.New(httpServer, grpcServer, signal.New())

	// Run servers.
	errs := func() []error {
		if errs := srv.Run(log); errs != nil {
			return errs
		}
		return srv.Wait()
	}()

	// Flush logger.
	_ = log.Sync()

	// Exit with the number of errors that occurred.
	os.Exit(len(errs))
}

// Setup HTTP handlers.
func setupHTTP(r *gin.Engine, tracer trace.Tracer, log *zap.Logger) {
	r.GET("/", func(c *gin.Context) {
		log := log.With(apmzap.TraceContext(c.Request.Context())...)
		log.Info("Foobar here")
		c.JSON(200, gin.H{
			"message": "Hello World",
		})
	})

	r.GET("/panic", func(c *gin.Context) {
		log := log.With(apmzap.TraceContext(c.Request.Context())...)
		log.Debug("Let's panic")
		panic("I have been asked to panic")
	})

	r.GET("/trace", func(c *gin.Context) {
		log := log.With(apmzap.TraceContext(c.Request.Context())...)
		time.Sleep(250)
		ctx, span := tracer.Start(c.Request.Context(), "foo")
		defer span.End()
		time.Sleep(250)
		span.AddEvent(ctx, "custom event")
		log.Info("Something happening right now")
		time.Sleep(250)
		func(ctx context.Context) {
			_, span := tracer.Start(ctx, "bar")
			defer span.End()
			time.Sleep(250)
			span.RecordError(ctx, errExample)
			log.Error("Bar eats foo")
			time.Sleep(250)
			span.SetAttribute("custom attribute", "foobar")
			time.Sleep(250)
		}(ctx)
		time.Sleep(250)
	})
}
