package server

import (
	"context"
	"fmt"
	"github.com/elskow/chef-infra/internal/api"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	config         *config.AppConfig
	log            *zap.Logger
	grpcServer     *grpc.Server
	authHandler    *auth.Handler
	authMiddleware *auth.AuthMiddleware
}

type Params struct {
	fx.In

	Config         *config.AppConfig
	Logger         *zap.Logger
	AuthHandler    *auth.Handler
	AuthMiddleware *auth.AuthMiddleware
}

func isProtectedEndpoint(method string) bool {
	isPublic, exists := api.PublicEndpoints[method]
	return !exists || !isPublic
}

func NewServer(p Params) *Server {
	authInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip authentication for non-protected endpoints
		if !isProtectedEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		// Authenticate the request
		newCtx, err := p.AuthMiddleware.AuthenticationMiddleware(ctx)
		if err != nil {
			p.Logger.Warn("authentication failed",
				zap.String("method", info.FullMethod),
				zap.Error(err))
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}

		// Call the handler with the authenticated context
		return handler(newCtx, req)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(authInterceptor),
		grpc.MaxRecvMsgSize(p.Config.GRPC.MaxReceiveMessageSize),
		grpc.MaxSendMsgSize(p.Config.GRPC.MaxSendMessageSize),
	}

	grpcServer := grpc.NewServer(opts...)

	server := &Server{
		config:         p.Config,
		log:            p.Logger,
		grpcServer:     grpcServer,
		authHandler:    p.AuthHandler,
		authMiddleware: p.AuthMiddleware,
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
