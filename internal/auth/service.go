package auth

import (
	"errors"
	"github.com/elskow/chef-infra/internal/config"
	"go.uber.org/zap"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	config     *config.AuthConfig
	log        *zap.Logger
	repository Repository
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewService(config *config.AuthConfig, log *zap.Logger, repo Repository) *Service {
	return &Service{
		config:     config,
		log:        log,
		repository: repo,
	}
}

func (s *Service) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *Service) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *Service) GenerateToken(username string) (string, error) {
	expirationTime := time.Now().Add(s.config.TokenExpiration)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (s *Service) RegisterUser(username, password, email string) error {
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return err
	}

	user := &User{
		Username:     username,
		PasswordHash: hashedPassword,
		Email:        email,
	}

	return s.repository.CreateUser(user)
}

func (s *Service) ValidateLogin(username, password string) (string, error) {
	user, err := s.repository.GetUserByUsername(username)
	if err != nil {
		if err == ErrUserNotFound {
			s.HashPassword("dummy") // Prevent timing attacks
			return "", ErrUserNotFound
		}
		return "", err
	}

	// Check if account is locked
	if user.Locked {
		if user.LockUntil != nil && time.Now().After(*user.LockUntil) {
			// Unlock account if lock duration has passed
			if err := s.repository.UnlockAccount(user.ID); err != nil {
				return "", err
			}
		} else {
			return "", errors.New("account is locked")
		}
	}

	if !s.CheckPasswordHash(password, user.PasswordHash) {
		// Increment failed login attempts
		if err := s.repository.UpdateLoginAttempts(user.ID, true); err != nil {
			s.log.Error("failed to update login attempts", zap.Error(err))
		}

		// Lock account after too many failed attempts
		if user.FailedLoginCount >= 5 {
			if err := s.repository.LockAccount(user.ID, time.Minute*15); err != nil {
				s.log.Error("failed to lock account", zap.Error(err))
			}
		}

		return "", ErrInvalidPassword
	}

	// Reset failed login attempts on successful login
	if err := s.repository.UpdateLoginAttempts(user.ID, false); err != nil {
		s.log.Error("failed to reset login attempts", zap.Error(err))
	}

	token, err := s.GenerateToken(user.Username)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) GenerateTokenPair(username string) (accessToken, refreshToken string, err error) {
	accessToken, err = s.GenerateToken(username)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = s.generateRefreshToken(username)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *Service) generateRefreshToken(username string) (string, error) {
	if !s.config.RefreshTokenEnabled {
		return "", errors.New("refresh token functionality is disabled")
	}

	// Refresh tokens have a longer expiration time
	expirationTime := time.Now().Add(s.config.TokenExpiration * 24) // 24x longer than access token
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "refresh",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *Service) RefreshToken(refreshToken string) (string, error) {
	// Validate refresh token and generate new access token
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	return s.GenerateToken(claims.Username)
}
