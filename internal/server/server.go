package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/elskow/chef-infra/proto/gen/echo"
)

type Server struct {
	config     *Config
	grpcServer *grpc.Server
	pb.UnimplementedGreeterServer
}

func NewServer(config *Config) *Server {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(config.GRPC.MaxReceiveMessageSize),
		grpc.MaxSendMsgSize(config.GRPC.MaxSendMessageSize),
	}

	grpcServer := grpc.NewServer(opts...)

	server := &Server{
		config:     config,
		grpcServer: grpcServer,
	}

	pb.RegisterGreeterServer(grpcServer, server)

	if config.GRPC.EnableReflection {
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

	log.Printf("gRPC server listening on %s", addr)
	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

func (s *Server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}
