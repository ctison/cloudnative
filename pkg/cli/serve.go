package cli

import (
	"os"

	"github.com/ctison/cloudnative/pkg/server"
	"github.com/ctison/cloudnative/pkg/server/http"
	"github.com/ctison/cloudnative/pkg/server/signal"
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
