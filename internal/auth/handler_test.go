package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/elskow/chef-infra/proto/gen/auth"
)

func TestHandler_Register(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		request  *pb.RegisterRequest
		setup    func() error
		wantCode codes.Code
		wantMsg  string
	}{
		{
			name: "duplicate username",
			request: &pb.RegisterRequest{
				Username: "duplicate",
				Password: "testpass123",
				Email:    "another@example.com",
			},
			setup: func() error {
				_, err := h.Register(ctx, &pb.RegisterRequest{
					Username: "duplicate",
					Password: "testpass123",
					Email:    "test@example.com",
				})
				return err
			},
			wantCode: codes.AlreadyExists,
			wantMsg:  "username already taken",
		},
		{
			name: "duplicate email",
			request: &pb.RegisterRequest{
				Username: "newuser",
				Password: "testpass123",
				Email:    "duplicate@example.com",
			},
			setup: func() error {
				_, err := h.Register(ctx, &pb.RegisterRequest{
					Username: "testuser",
					Password: "testpass123",
					Email:    "duplicate@example.com",
				})
				return err
			},
			wantCode: codes.AlreadyExists,
			wantMsg:  "email already registered",
		},
		{
			name: "empty username",
			request: &pb.RegisterRequest{
				Password: "testpass123",
				Email:    "test@example.com",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty password",
			request: &pb.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty email",
			request: &pb.RegisterRequest{
				Username: "testuser",
				Password: "testpass123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid email format",
			request: &pb.RegisterRequest{
				Username: "testuser",
				Password: "testpass123",
				Email:    "invalid-email",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "username too short",
			request: &pb.RegisterRequest{
				Username: "ab",
				Password: "testpass123",
				Email:    "test@example.com",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "username too long",
			request: &pb.RegisterRequest{
				Username: "thisusernameiswaytoolongandshouldfail",
				Password: "testpass123",
				Email:    "test@example.com",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "password too short",
			request: &pb.RegisterRequest{
				Username: "testuser",
				Password: "short",
				Email:    "test@example.com",
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run setup if provided
			if tt.setup != nil {
				setupErr := tt.setup()
				require.NoError(t, setupErr, "Setup should succeed")
			}

			resp, err := h.Register(ctx, tt.request)

			if tt.wantCode != codes.OK {
				require.Error(t, err, "Expected an error")
				st, ok := status.FromError(err)
				require.True(t, ok, "Error should be a status error")
				assert.Equal(t, tt.wantCode, st.Code(), "Status code should match")
				if tt.wantMsg != "" {
					assert.Contains(t, st.Message(), tt.wantMsg, "Error message should match")
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, resp.Success)
			assert.NotEmpty(t, resp.Message)

			// Verify user was created
			user, err := h.service.repository.GetUserByUsername(tt.request.Username)
			require.NoError(t, err)
			assert.Equal(t, tt.request.Username, user.Username)
			assert.Equal(t, tt.request.Email, user.Email)
			assert.NotEqual(t, tt.request.Password, user.PasswordHash)
		})
	}
}

func TestHandler_Login(t *testing.T) {
	repo := newMockRepository()
	svc := newTestServiceWithRepo(t, repo)
	h := NewHandler(svc, newTestLogger(t))
	ctx := context.Background()

	registerResp, err := h.Register(ctx, &pb.RegisterRequest{
		Username: "testuser",
		Password: "testpass123",
		Email:    "test@example.com",
	})
	require.NoError(t, err)
	require.True(t, registerResp.Success)

	tests := []struct {
		name     string
		request  *pb.LoginRequest
		wantCode codes.Code
	}{
		{
			name: "valid credentials",
			request: &pb.LoginRequest{
				Username: "testuser",
				Password: "testpass123",
			},
			wantCode: codes.OK,
		},
		{
			name: "wrong password",
			request: &pb.LoginRequest{
				Username: "testuser",
				Password: "wrongpass",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "nonexistent user",
			request: &pb.LoginRequest{
				Username: "nonexistent",
				Password: "testpass123",
			},
			wantCode: codes.NotFound,
		},
		{
			name: "missing username",
			request: &pb.LoginRequest{
				Password: "testpass123",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing password",
			request: &pb.LoginRequest{
				Username: "testuser",
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := h.Login(ctx, tt.request)

			if tt.wantCode != codes.OK {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, status.Code(err))
				return
			}

			require.NoError(t, err)
			assert.True(t, resp.Success)
			assert.NotEmpty(t, resp.AccessToken)
			assert.NotEmpty(t, resp.RefreshToken)
			assert.NotEmpty(t, resp.Message)
		})
	}
}

func TestHandler_ValidateToken(t *testing.T) {
	repo := newMockRepository()
	svc := newTestServiceWithRepo(t, repo)
	h := NewHandler(svc, newTestLogger(t))
	ctx := context.Background()

	_, err := h.Register(ctx, &pb.RegisterRequest{
		Username: "testuser",
		Password: "testpass123",
		Email:    "test@example.com",
	})
	require.NoError(t, err)

	loginResp, err := h.Login(ctx, &pb.LoginRequest{
		Username: "testuser",
		Password: "testpass123",
	})
	require.NoError(t, err)
	validToken := loginResp.AccessToken

	tests := []struct {
		name      string
		request   *pb.ValidateTokenRequest
		wantValid bool
	}{
		{
			name: "valid token",
			request: &pb.ValidateTokenRequest{
				Token: validToken,
			},
			wantValid: true,
		},
		{
			name: "invalid token",
			request: &pb.ValidateTokenRequest{
				Token: "invalid.token.here",
			},
			wantValid: false,
		},
		{
			name: "empty token",
			request: &pb.ValidateTokenRequest{
				Token: "",
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := h.ValidateToken(ctx, tt.request)
			require.NoError(t, err)

			assert.Equal(t, tt.wantValid, resp.Valid)
			if tt.wantValid {
				assert.NotEmpty(t, resp.Username)
				assert.Equal(t, "testuser", resp.Username)
			}
		})
	}
}
