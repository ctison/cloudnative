package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	// We run servers in another function so that deferred function calls will be honored before we call os.Exit.
	errCount := runServers(log)

	// Flush logger here and not in a deferred function calls because we use os.Exit.
	_ = log.Sync()

	// We exit with the number of errors that occurred while running servers.
	os.Exit(errCount)
}

// Start servers and return the errors' count (which may be 0) when all servers are (gracefully) shutdowned.
func runServers(log *zap.Logger) (errCount int) {
	// ServerFunc starts an asynchronous server and returns a function to stop that server asynchronously.
	// If this function returns no error, it is expected that it will send two errors through the errs channel:
	// - One from the server
	// - One from its stop function
	type ServerFunc func(_ *zap.Logger, errs chan<- error) (func(), error)

	// List the servers this program will run.
	servers := []ServerFunc{
		signalServer,
		httpServer,
	}

	// Make an array to gather servers' stop functions.
	stops := make([]func(), 0, len(servers))

	// Make a channel of errors that servers may use to stop the program.
	errs := make(chan error, len(servers)*2)

	// Start the servers.
	for _, server := range servers {
		stop, err := server(log.Named("server"), errs)
		if err != nil {
			errCount++
			errCount += stopServers(stops, errs, len(stops)*2)
			return errCount
		}
		stops = append(stops, stop)
	}

	// Wait until a server send an error (which may be nil).
	if err := <-errs; err != nil {
		errCount++
	}

	// Gracefully shutdown servers and return.
	errCount += stopServers(stops, errs, len(stops)*2-1)
	return errCount
}

// Gracefully shutdown servers after having received waitFor errors from errs.
func stopServers(stops []func(), errs <-chan error, waitFor int) (errCount int) {
	for _, stop := range stops {
		stop()
	}
	for i := 0; i < waitFor; i++ {
		if err := <-errs; err != nil {
			errCount++
		}
	}
	return
}

// Start handling signals and send a nil error to errs when receiving one.
func signalServer(log *zap.Logger, errs chan<- error) (func(), error) {
	// Add contextual prefix to the logger.
	log = log.Named("signals")

	// Setup signals channel.
	signals := make(chan os.Signal, 1)
	signalsNumber := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	signal.Notify(signals, signalsNumber...)

	// Instantiate a stop channel to stop handling signals.
	stop := make(chan struct{}, 1)

	// Start handling signals.
	go func() {
		log.Info(fmt.Sprintf("start handling: %v", signalsNumber))
		select {
		case <-stop:
			break
		case <-signals:
			break
		}
		signal.Stop(signals)
		log.Info("shutdowned")
		errs <- nil
	}()

	// Return stop function.
	return func() {
		stop <- struct{}{}
		errs <- nil
	}, nil
}

// Start an HTTP server.
func httpServer(log *zap.Logger, errs chan<- error) (func(), error) {
	// Add contextual prefix to the logger.
	log = log.Named("http")

	// Any error is semantically wrapped.
	wrapError := func(err error) error {
		return fmt.Errorf("http server: %w", err)
	}

	// Instantiate multiplexer - you can use alternatives like https://github.com/gorilla/mux.
	mux := http.NewServeMux()

	// Setup probes handlers
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ready"}`)
	})
	mux.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"alive"}`)
	})

	// Setup your business logic handlers.
	// ...
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"msg":"Hello World"}`)
	})

	// Instantiate an HTTP server.
	httpServer := &http.Server{Addr: ":8080", Handler: mux}

	// Start the HTTP server.
	go func() {
		log.Info("start listening on :8080")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(err.Error())
			errs <- wrapError(err)
			return
		}
		errs <- nil
	}()

	// Return the stop function.
	return func() {
		if err := httpServer.Shutdown(context.Background()); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				errs <- nil
				return
			}
			log.Error(err.Error())
			errs <- wrapError(err)
		} else {
			log.Info("shutdowned")
			errs <- nil
		}
	}, nil
}
