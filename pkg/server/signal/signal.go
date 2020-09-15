package signal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

type Server struct {
	Signals []os.Signal
}

func New(signals ...os.Signal) *Server {
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	return &Server{
		Signals: signals,
	}
}

func (srv *Server) Start(ctx context.Context, log *zap.Logger, errs chan<- error) error {
	log = log.Named("signal")
	c := make(chan os.Signal, 1)
	signal.Notify(c, srv.Signals...)
	log.Info(fmt.Sprintf("started handling os signals: %v", srv.Signals))

	go func() {
		select {
		case <-ctx.Done():
			break
		case sig := <-c:
			log.Info("received signal", zap.String("signal", sig.String()))
		}
		signal.Stop(c)
		errs <- nil
		errs <- nil
	}()

	return nil
}
