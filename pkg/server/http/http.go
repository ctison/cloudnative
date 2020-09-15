package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Server struct {
	server *http.Server
}

func New(server *http.Server) *Server {
	r := mux.NewRouter()
	if server == nil {
		server = &http.Server{Addr: ":8080"}
	}
	server.Handler = r
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ready"}`)
	})
	r.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"alive"}`)
	})
	return &Server{
		server: server,
	}
}

func (srv *Server) Mux() *mux.Router {
	return srv.server.Handler.(*mux.Router)
}

func (srv *Server) Start(ctx context.Context, log *zap.Logger, errs chan<- error) error {
	// Add context to the logger.
	log = log.Named("http")

	// Any error is semantically wrapped.
	wrapError := func(err error) error {
		return fmt.Errorf("http server: %w", err)
	}

	// Start the HTTP server.
	go func() {
		log.Info("start listening on :8080")
		if err := srv.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(err.Error())
			errs <- wrapError(err)
			return
		}
		errs <- nil
	}()

	// Wait for the context cancellation to shutdown the server.
	go func() {
		<-ctx.Done()
		if err := srv.server.Shutdown(context.Background()); err != nil {
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
