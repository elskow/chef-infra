package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_HashPassword(t *testing.T) {
	svc := newTestService(t)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "testpassword123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // bcrypt handles empty passwords
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := svc.HashPassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, hash)

			// Verify the hash
			valid := svc.CheckPasswordHash(tt.password, hash)
			assert.True(t, valid)
		})
	}
}

func TestService_GenerateToken(t *testing.T) {
	svc := newTestService(t)

	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{
			name:     "valid username",
			username: "testuser",
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			wantErr:  false, // JWT handles empty claims
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.GenerateToken(tt.username)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Validate the token
			claims, err := svc.ValidateToken(token)
			assert.NoError(t, err)
			assert.Equal(t, tt.username, claims.Username)
		})
	}
}

func TestService_ValidateToken(t *testing.T) {
	svc := newTestService(t)

	tests := []struct {
		name       string
		setupToken func() string
		wantErr    bool
		wantUser   string
	}{
		{
			name: "valid token",
			setupToken: func() string {
				token, _ := svc.GenerateToken("testuser")
				return token
			},
			wantErr:  false,
			wantUser: "testuser",
		},
		{
			name: "expired token",
			setupToken: func() string {
				// Create service with expired token
				expiredConfig := newTestConfig()
				expiredConfig.TokenExpiration = -time.Hour
				expiredSvc := NewService(expiredConfig, newTestLogger(t))
				token, _ := expiredSvc.GenerateToken("testuser")
				return token
			},
			wantErr:  true,
			wantUser: "",
		},
		{
			name: "invalid token",
			setupToken: func() string {
				return "invalid.token.here"
			},
			wantErr:  true,
			wantUser: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupToken()
			claims, err := svc.ValidateToken(token)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantUser, claims.Username)
		})
	}
}
