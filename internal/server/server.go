package server

import (
	"fmt"
	"go.uber.org/zap/zapcore"
	"net"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/elskow/chef-infra/internal/auth"
	"github.com/elskow/chef-infra/internal/config"
	pb "github.com/elskow/chef-infra/proto/gen/auth"
)

type Server struct {
	config      *config.AppConfig
	log         *zap.Logger
	grpcServer  *grpc.Server
	authHandler *auth.Handler
}

type Params struct {
	fx.In

	Config      *config.AppConfig
	Logger      *zap.Logger
	AuthHandler *auth.Handler
}

func NewServer(p Params) *Server {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(p.Config.GRPC.MaxReceiveMessageSize),
		grpc.MaxSendMsgSize(p.Config.GRPC.MaxSendMessageSize),
	}

	grpcServer := grpc.NewServer(opts...)

	server := &Server{
		config:      p.Config,
		log:         p.Logger,
		grpcServer:  grpcServer,
		authHandler: p.AuthHandler,
	}

	// Register services
	pb.RegisterAuthServer(grpcServer, p.AuthHandler)

	if p.Config.GRPC.EnableReflection {
		reflection.Register(grpcServer)
	}

	return server
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.log.Info("Starting gRPC server",
		zap.String("address", addr),
		zap.Object("config", serverConfigToField(s.config)),
	)

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func serverConfigToField(config *config.AppConfig) zapcore.ObjectMarshaler {
	return zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("environment", os.Getenv("APP_ENV"))
		enc.AddBool("reflection_enabled", config.GRPC.EnableReflection)
		enc.AddInt("max_receive_size", config.GRPC.MaxReceiveMessageSize)
		enc.AddInt("max_send_size", config.GRPC.MaxSendMessageSize)
		return nil
	})
}

func (s *Server) Stop() {
	s.log.Info("shutting down gRPC server")
	s.grpcServer.GracefulStop()
}
