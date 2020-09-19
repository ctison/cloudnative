package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/ctison/cloudnative/pkg/server/grpc/api"
	grpcotel "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc"
	otel "go.opentelemetry.io/otel/api/global"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	server *grpc.Server
	log    *zap.Logger
}

type fooServer struct {
	api.UnimplementedFooServer
}

func (srv *fooServer) Trace(ctx context.Context, r *api.TraceRequest) (*api.TraceResponse, error) {
	return &api.TraceResponse{
		Msg: r.Bar + r.Baz,
	}, nil
}

func New(log *zap.Logger) *Server {
	server := grpc.NewServer(
		// Only one interceptor can be set.
		// grpc.UnaryInterceptor(apmgrpc.NewUnaryServerInterceptor(apmgrpc.WithRecovery())),
		grpc.UnaryInterceptor(grpcotel.UnaryServerInterceptor(otel.Tracer(""))),
		grpc.StreamInterceptor(grpcotel.StreamServerInterceptor(otel.Tracer(""))),
	)
	api.RegisterFooServer(server, &fooServer{})
	return &Server{
		server: server,
		log:    log,
	}
}

func (srv *Server) Start(ctx context.Context, log *zap.Logger, errs chan<- error) error {
	log = log.Named("grpc")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return fmt.Errorf(`failed to listen on ":50051": %w`, err)
	}

	// Start gRPC server.
	go func() {
		log.Info("start listensing on :50051")
		if err := srv.server.Serve(lis); err != nil {
			log.Error(err.Error())
			errs <- fmt.Errorf("grpc server: %w", err)
		}
		log.Info("shutdown")
		errs <- nil
	}()

	// Handle termination.
	go func() {
		<-ctx.Done()
		srv.server.GracefulStop()
		errs <- nil
	}()

	return nil
}
