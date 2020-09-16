package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ctison/cloudnative/pkg/server"
	serverHTTP "github.com/ctison/cloudnative/pkg/server/http"
	"github.com/ctison/cloudnative/pkg/server/signal"
	"github.com/gin-gonic/gin"

	"go.uber.org/zap"
)

// Version can be overridden by go linker: `go build -ldflags '-X main.version=v1.2.3'`.
var version = "v0.0.0"

func main() {
	// Setup the command-line interface (CLI).
	printHelp := flag.Bool("h", false, "Print help and exit")
	printVersion := flag.Bool("v", false, "Print version and exit")
	logLevel := zap.LevelFlag("l", zap.InfoLevel, "Configure log level")
	devMode := flag.Bool("d", false, "Enable more loggin with dev mode")
	flag.Parse()

	if *printHelp {
		fmt.Printf("Usage: %s [OPTIONS]\n\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *printVersion {
		fmt.Printf("version: %s\n", version)
		os.Exit(0)
	}

	// Instantiate logger.
	logConfig := zap.NewProductionConfig()
	logConfig.Level.SetLevel(*logLevel)
	logConfig.Development = *devMode
	log, err := logConfig.Build()
	if err != nil {
		// Using panic here is a sane behavior because no servers are running and logging cannot happen.
		panic(err)
	}

	errs := run(log, *devMode)

	// Flush logger here and not in a deferred function calls because we use os.Exit.
	_ = log.Sync()

	// We exit with the number of errors that occurred while running servers.
	os.Exit(len(errs))
}

func run(log *zap.Logger, devMode bool) []error {
	httpServer := serverHTTP.New(log, devMode, nil)
	r := httpServer.Gin()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World",
		})
	})
	srv := server.New(httpServer, signal.New())

	if errs := srv.Run(log); errs != nil {
		return errs
	}

	return srv.Wait()
}
