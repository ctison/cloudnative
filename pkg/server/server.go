package server

import (
	"context"

	"go.uber.org/zap"
)

type Server interface {
	Start(context.Context, *zap.Logger, chan<- error) error
}

type Servers struct {
	servers []Server
	cancels []context.CancelFunc
	errs    chan error
}

func New(servers ...Server) *Servers {
	if len(servers) == 0 {
		panic("need at least one server")
	}
	return &Servers{
		servers: servers,
	}
}

func (srv *Servers) Run(log *zap.Logger) []error {
	// Do nothing if the servers are already running.
	if len(srv.cancels) > 0 {
		return nil
	}

	srv.cancels = make([]context.CancelFunc, 0, len(srv.servers))
	srv.errs = make(chan error, len(srv.servers))

	var errs []error

	// Start the servers.
	for _, server := range srv.servers {
		ctx, cancel := context.WithCancel(context.Background())
		if err := server.Start(ctx, log.Named("server"), srv.errs); err != nil {
			errs = append(errs, err)
			cancel()
			return append(errs, srv.Stop()...)
		}
		srv.cancels = append(srv.cancels, cancel)
	}

	return nil
}

func (srv *Servers) Stop() []error {
	return srv.stop(len(srv.cancels) * 2)
}

func (srv *Servers) Wait() []error {
	if len(srv.cancels) == 0 {
		panic("servers are not running")
	}
	var errs []error
	if err := <-srv.errs; err != nil {
		errs = append(errs, err)
	}
	if errs2 := srv.stop(len(srv.cancels)*2 - 1); errs2 != nil {
		errs = append(errs, errs2...)
	}
	return errs
}

func (srv *Servers) stop(waitFor int) []error {
	if waitFor <= 0 {
		panic("waitFor must be positive")
	}
	for _, cancel := range srv.cancels {
		cancel()
	}
	var errs []error
	for i := 0; i < waitFor; i++ {
		if err := <-srv.errs; err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
