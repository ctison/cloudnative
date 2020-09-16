package cli

import (
	"context"
	"os"
	"time"

	otelGlobal "go.opentelemetry.io/otel/api/global"

	"github.com/ctison/cloudnative/pkg/server"
	"github.com/ctison/cloudnative/pkg/server/http"
	"github.com/ctison/cloudnative/pkg/server/signal"
	"github.com/ctison/cloudnative/pkg/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func (cli *CLI) Serve(cmd *cobra.Command, args []string) {
	// Instantiate logger.
	logConfig := zap.NewProductionConfig()
	logConfig.Level.SetLevel(cli.serve.logLevel.Level)
	logConfig.Development = cli.serve.devMode
	log, err := logConfig.Build()
	if err != nil {
		panic(err)
	}

	// Setup telemetry.
	if err := telemetry.Init(); err != nil {
		log.Error("failed to start telemetry", zap.Error(err))
		os.Exit(1)
	}

	// Instantiate http server.
	httpServer := http.New(log, cli.serve.devMode, nil)
	r := httpServer.Gin()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World",
		})
	})

	r.GET("/panic", func(c *gin.Context) {
		panic("I have been asked to panic")
	})

	r.GET("/trace", func(c *gin.Context) {
		tracer := otelGlobal.Tracer("xXx")
		ctx, span := tracer.Start(context.Background(), "job")
		defer span.End()
		time.Sleep(1 * time.Second)
		span.AddEvent(ctx, "pip")
		func(ctx context.Context) {
			ctx, span := tracer.Start(ctx, "subjob")
			time.Sleep(time.Second)
			span.AddEvent(ctx, "pop")
			defer span.End()
		}(ctx)
	})

	// Instantiate all servers.
	srv := server.New(httpServer, signal.New())

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
