package auth

import (
	"errors"
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
				expiredConfig.TokenExpiration = -time.Hour
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

func TestService_ValidateLogin_AccountLocking(t *testing.T) {
	svc := newTestService(t)
	username := "testuser"
	password := "testpass123"
	email := "test@example.com"

	// Register test user
	err := svc.RegisterUser(username, password, email)
	require.NoError(t, err)

	tests := []struct {
		name          string
		attempts      int
		waitDuration  time.Duration
		finalPassword string
		wantErr       error
	}{
		{
			name:          "successful login",
			attempts:      0,
			finalPassword: password,
			wantErr:       nil,
		},
		{
			name:          "account locks after 5 failed attempts",
			attempts:      5,
			finalPassword: password,
			wantErr:       errors.New("account is locked"),
		},
		{
			name:          "wrong password fails",
			attempts:      1,
			finalPassword: "wrongpass",
			wantErr:       ErrInvalidPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the service for each test
			svc = newTestService(t)
			err := svc.RegisterUser(username, password, email)
			require.NoError(t, err)

			// Simulate failed login attempts
			for i := 0; i < tt.attempts; i++ {
				_, err := svc.ValidateLogin(username, "wrongpass")
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidPassword)
			}

			if tt.waitDuration > 0 {
				time.Sleep(tt.waitDuration)
			}

			// Try final login
			token, err := svc.ValidateLogin(username, tt.finalPassword)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, token)
		})
	}
}

func TestService_ValidateLogin_AccountUnlocking(t *testing.T) {
	svc := newTestService(t)
	username := "testuser"
	password := "testpass123"
	email := "test@example.com"

	// Register test user
	err := svc.RegisterUser(username, password, email)
	require.NoError(t, err)

	// Lock account with failed attempts
	for i := 0; i < 5; i++ {
		_, err := svc.ValidateLogin(username, "wrongpass")
		require.Error(t, err)
	}

	// Verify account is locked
	_, err = svc.ValidateLogin(username, password)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account is locked")

	// Get user and manually set lock time to the past
	user, err := svc.repository.GetUserByUsername(username)
	require.NoError(t, err)
	pastTime := time.Now().Add(-time.Hour)
	user.LockUntil = &pastTime

	// Try login again - should succeed now
	token, err := svc.ValidateLogin(username, password)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify account is unlocked
	user, err = svc.repository.GetUserByUsername(username)
	require.NoError(t, err)
	assert.False(t, user.Locked)
	assert.Zero(t, user.FailedLoginCount)
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
				cfg.TokenExpiration = -time.Hour
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
