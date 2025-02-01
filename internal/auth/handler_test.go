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

func newTestHandler(t *testing.T) *Handler {
	return NewHandler(newTestService(t), newTestLogger(t))
}

func TestHandler_Register(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		request  *pb.RegisterRequest
		wantCode codes.Code
	}{
		{
			name: "valid registration",
			request: &pb.RegisterRequest{
				Username: "testuser",
				Password: "testpass123",
				Email:    "test@example.com",
			},
			wantCode: codes.OK,
		},
		{
			name: "empty password",
			request: &pb.RegisterRequest{
				Username: "testuser",
				Password: "",
				Email:    "test@example.com",
			},
			wantCode: codes.Internal, // because hash will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := h.Register(ctx, tt.request)

			if tt.wantCode != codes.OK {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, status.Code(err))
				return
			}

			require.NoError(t, err)
			assert.True(t, resp.Success)
			assert.NotEmpty(t, resp.Message)
		})
	}
}

func TestHandler_Login(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()

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
			assert.NotEmpty(t, resp.Token)
			assert.NotEmpty(t, resp.Message)
		})
	}
}

func TestHandler_ValidateToken(t *testing.T) {
	h := newTestHandler(t)
	ctx := context.Background()

	// First generate a valid token
	loginResp, err := h.Login(ctx, &pb.LoginRequest{
		Username: "testuser",
		Password: "testpass123",
	})
	require.NoError(t, err)
	validToken := loginResp.Token

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
			require.NoError(t, err) // ValidateToken doesn't return errors, only invalid status

			assert.Equal(t, tt.wantValid, resp.Valid)
			if tt.wantValid {
				assert.NotEmpty(t, resp.Username)
			}
		})
	}
}
