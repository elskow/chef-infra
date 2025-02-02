package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elskow/chef-infra/internal/config"
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
				expiredConfig.AccessTokenDuration = -time.Hour
				expiredSvc := NewService(
					expiredConfig,
					newTestLogger(t),
					newMockRepository(),
				)
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

func TestService_RegisterUser(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		email    string
		setup    func(*Service)
		wantErr  error
	}{
		{
			name:     "successful registration",
			username: "testuser",
			password: "testpass123",
			email:    "test@example.com",
			wantErr:  nil,
		},
		{
			name:     "duplicate username",
			username: "existing",
			password: "testpass123",
			email:    "new@example.com",
			setup: func(s *Service) {
				_ = s.RegisterUser("existing", "pass123", "test@example.com")
			},
			wantErr: ErrUserExists,
		},
		{
			name:     "duplicate email",
			username: "newuser",
			password: "testpass123",
			email:    "existing@example.com",
			setup: func(s *Service) {
				_ = s.RegisterUser("testuser", "pass123", "existing@example.com")
			},
			wantErr: ErrUserExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t)
			if tt.setup != nil {
				tt.setup(svc)
			}

			err := svc.RegisterUser(tt.username, tt.password, tt.email)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)

			// Verify user was created correctly
			user, err := svc.repository.GetUserByUsername(tt.username)
			require.NoError(t, err)
			assert.Equal(t, tt.email, user.Email)
			assert.True(t, svc.CheckPasswordHash(tt.password, user.PasswordHash))
		})
	}
}

func TestService_GenerateTokenPair(t *testing.T) {
	svc := newTestService(t)
	username := "testuser"

	tests := []struct {
		name          string
		setupConfig   func(*config.AuthConfig)
		wantErr       bool
		validateToken bool
	}{
		{
			name:          "successful token pair generation",
			validateToken: true,
		},
		{
			name: "refresh token disabled",
			setupConfig: func(cfg *config.AuthConfig) {
				cfg.RefreshTokenEnabled = false
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupConfig != nil {
				tt.setupConfig(svc.config)
			}

			accessToken, refreshToken, err := svc.GenerateTokenPair(username)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, accessToken)
			assert.NotEmpty(t, refreshToken)

			if tt.validateToken {
				// Validate access token
				claims, err := svc.ValidateToken(accessToken)
				require.NoError(t, err)
				assert.Equal(t, username, claims.Username)

				// Validate refresh token
				claims, err = svc.ValidateToken(refreshToken)
				require.NoError(t, err)
				assert.Equal(t, username, claims.Username)
				assert.Equal(t, "refresh", claims.Subject)
			}
		})
	}
}

func TestService_RefreshToken(t *testing.T) {
	svc := newTestService(t)
	username := "testuser"

	// Generate initial token pair
	_, refreshToken, err := svc.GenerateTokenPair(username)
	require.NoError(t, err)

	tests := []struct {
		name       string
		token      string
		wantErr    bool
		setupToken func() string
	}{
		{
			name:  "valid refresh token",
			token: refreshToken,
		},
		{
			name:    "invalid refresh token",
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name: "expired refresh token",
			setupToken: func() string {
				cfg := newTestConfig()
				cfg.RefreshTokenDuration = -time.Hour
				expiredSvc := NewService(cfg, newTestLogger(t), newMockRepository())
				_, refresh, _ := expiredSvc.GenerateTokenPair(username)
				return refresh
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenToUse := tt.token
			if tt.setupToken != nil {
				tokenToUse = tt.setupToken()
			}

			newToken, err := svc.RefreshToken(tokenToUse)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, newToken)

			// Validate new token
			claims, err := svc.ValidateToken(newToken)
			require.NoError(t, err)
			assert.Equal(t, username, claims.Username)
		})
	}
}

func TestService_CheckPasswordHash(t *testing.T) {
	svc := newTestService(t)

	tests := []struct {
		name        string
		password    string
		hash        string
		wantMatches bool
	}{
		{
			name:        "matching password",
			password:    "testpass123",
			wantMatches: true,
		},
		{
			name:        "non-matching password",
			password:    "wrongpass",
			wantMatches: false,
		},
		{
			name:        "empty password",
			password:    "",
			wantMatches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := svc.HashPassword("testpass123")
			require.NoError(t, err)

			matches := svc.CheckPasswordHash(tt.password, hash)
			assert.Equal(t, tt.wantMatches, matches)
		})
	}
}
