package auth

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/elskow/chef-infra/proto/gen/auth"
)

type Handler struct {
	pb.UnimplementedAuthServer
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

func (h *Handler) Register(_ context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	h.log.Info("handling register request", zap.String("username", req.Username))

	hashedPassword, err := h.service.HashPassword(req.Password)
	if err != nil {
		h.log.Error("failed to hash password", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	// TODO: Database logic to store user details
	_ = hashedPassword

	return &pb.RegisterResponse{
		Success: true,
		Message: "User registered successfully",
	}, nil
}

func (h *Handler) Login(_ context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// TODO: Database logic to check if user exists
	// For now, using dummy validation
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password required")
	}

	// TODO: Database logic to fetch user details

	token, err := h.service.GenerateToken(req.Username)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb.LoginResponse{
		Success: true,
		Token:   token,
		Message: "Login successful",
	}, nil
}

func (h *Handler) ValidateToken(_ context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	claims, err := h.service.ValidateToken(req.Token)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: err.Error(),
		}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:    true,
		Username: claims.Username,
		Message:  "Token is valid",
	}, nil
}
