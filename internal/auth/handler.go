package auth

import (
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/mail"

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
	// Validate input fields
	if err := validateRegisterRequest(req); err != nil {
		h.log.Warn("invalid register request",
			zap.String("error", err.Error()),
			zap.Any("request", req))
		return nil, err
	}

	h.log.Info("handling register request", zap.String("username", req.Username))

	// Check if user already exists
	if _, err := h.service.repository.GetUserByUsername(req.Username); err == nil {
		return nil, status.Error(codes.AlreadyExists, "username already taken")
	}

	// Check if email already exists
	if _, err := h.service.repository.GetUserByEmail(req.Email); err == nil {
		return nil, status.Error(codes.AlreadyExists, "email already registered")
	}

	// Register the user
	if err := h.service.RegisterUser(req.Username, req.Password, req.Email); err != nil {
		if err == ErrUserExists {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		h.log.Error("failed to register user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to register user")
	}

	return &pb.RegisterResponse{
		Success: true,
		Message: "User registered successfully",
	}, nil
}

func (h *Handler) Login(_ context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Validate input fields
	if err := validateLoginRequest(req); err != nil {
		h.log.Warn("invalid login request",
			zap.String("error", err.Error()),
			zap.String("username", req.Username))
		return nil, err
	}

	// Validate credentials and generate token
	token, err := h.service.ValidateLogin(req.Username, req.Password)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		if err == ErrInvalidPassword {
			return nil, status.Error(codes.Unauthenticated, "invalid password")
		}
		h.log.Error("login failed",
			zap.String("username", req.Username),
			zap.Error(err))
		return nil, status.Error(codes.Internal, "login failed")
	}

	return &pb.LoginResponse{
		Success: true,
		Token:   token,
		Message: "Login successful",
	}, nil
}

func (h *Handler) ValidateToken(_ context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: "token is required",
		}, nil
	}

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

func validateRegisterRequest(req *pb.RegisterRequest) error {
	if req.Username == "" {
		return status.Error(codes.InvalidArgument, "username is required")
	}
	if len(req.Username) < 3 || len(req.Username) > 32 {
		return status.Error(codes.InvalidArgument, "username must be between 3 and 32 characters")
	}
	if req.Password == "" {
		return status.Error(codes.InvalidArgument, "password is required")
	}
	if len(req.Password) < 8 {
		return status.Error(codes.InvalidArgument, "password must be at least 8 characters")
	}
	if req.Email == "" {
		return status.Error(codes.InvalidArgument, "email is required")
	}
	if !isValidEmail(req.Email) {
		return status.Error(codes.InvalidArgument, "invalid email format")
	}
	return nil
}

func validateLoginRequest(req *pb.LoginRequest) error {
	if req.Username == "" {
		return status.Error(codes.InvalidArgument, "username is required")
	}
	if req.Password == "" {
		return status.Error(codes.InvalidArgument, "password is required")
	}
	return nil
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
