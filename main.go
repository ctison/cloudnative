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

// Start servers and return the errors' count (which may be 0) when all servers are (gracefully) shutdown.
func runServers(log *zap.Logger) (errCount int) {
	// ServerFunc starts an asynchronous server or returns an error.
	// When the context is canceled, two errors (possibly nil) are expected from the errors' channel:
	// - One from the server.
	// - One from its stop procedure.
	type ServerFunc func(context.Context, *zap.Logger, chan<- error) error

	// List the servers this program will run.
	servers := []ServerFunc{
		signalServer,
		httpServer,
	}

	// Make an array to gather servers' cancel functions.
	cancels := make([]context.CancelFunc, 0, len(servers))

	// Make a channel of errors that servers use to pass errors back to the main thread.
	errs := make(chan error, len(servers))

	// Start the servers.
	for _, server := range servers {
		ctx, cancel := context.WithCancel(context.Background())
		if err := server(ctx, log.Named("server"), errs); err != nil {
			cancel()
			errCount++
			errCount += stopServers(cancels, errs, len(cancels)*2)
			return errCount
		}
		cancels = append(cancels, cancel)
	}

	// Wait until a server sends an error.
	if err := <-errs; err != nil {
		errCount++
	}

	// Gracefully shutdown servers and return.
	errCount += stopServers(cancels, errs, len(cancels)*2-1)
	return errCount
}

// Gracefully shutdown servers after having received waitFor errors from errs.
func stopServers(cancels []context.CancelFunc, errs <-chan error, waitFor int) (errCount int) {
	for _, cancel := range cancels {
		cancel()
	}
	for i := 0; i < waitFor; i++ {
		if err := <-errs; err != nil {
			errCount++
		}
	}
	return
}

// Start handling signals and send a nil error to errs when receiving one.
func signalServer(ctx context.Context, log *zap.Logger, errs chan<- error) error {
	// Add contextual prefix to the logger.
	log = log.Named("signals")

	// Setup signals channel.
	signals := make(chan os.Signal, 1)
	signalsNumber := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	signal.Notify(signals, signalsNumber...)

	// Start handling signals.
	go func() {
		log.Info(fmt.Sprintf("start handling: %v", signalsNumber))
		select {
		case <-ctx.Done():
			break
		case sig := <-signals:
			log.Info(fmt.Sprintf("received signal: %v", sig))
		}
		signal.Stop(signals)
		log.Info("shutdown")
		errs <- nil
		errs <- nil
	}()

	return nil
}

// Start an HTTP server.
func httpServer(ctx context.Context, log *zap.Logger, errs chan<- error) error {
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

	// Wait for the context cancellation to shutdown the server.
	go func() {
		<-ctx.Done()
		if err := httpServer.Shutdown(context.Background()); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				errs <- nil
				return
			}
			log.Error(err.Error())
			errs <- wrapError(err)
		} else {
			log.Info("shutdown")
			errs <- nil
		}
	}()

	return nil
}
