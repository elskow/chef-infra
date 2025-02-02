package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/elskow/chef-infra/internal/config"
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
	expirationTime := time.Now().Add(s.config.AccessTokenDuration)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "access",
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

func (s *Service) validateTokenType(claims *Claims, expectedType string) error {
	if claims.Subject != expectedType {
		return fmt.Errorf("invalid token type: expected %s, got %s", expectedType, claims.Subject)
	}
	return nil
}

func (s *Service) ValidateLoginWithRefresh(username, password string) (accessToken, refreshToken string, err error) {
	user, err := s.repository.GetUserByUsername(username)
	if err != nil {
		if err == ErrUserNotFound {
			s.HashPassword("dummy") // Prevent timing attacks
			return "", "", ErrUserNotFound
		}
		return "", "", err
	}

	if !s.CheckPasswordHash(password, user.PasswordHash) {
		return "", "", ErrInvalidPassword
	}

	// Generate token pair
	return s.GenerateTokenPair(user.Username)
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

	if !s.CheckPasswordHash(password, user.PasswordHash) {
		return "", ErrInvalidPassword
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

	expirationTime := time.Now().Add(s.config.RefreshTokenDuration) // Use RefreshTokenDuration
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
	// Validate refresh token
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	// Ensure it's a refresh token
	if err := s.validateTokenType(claims, "refresh"); err != nil {
		return "", err
	}

	// Generate new access token
	return s.GenerateToken(claims.Username)
}

func (s *Service) RefreshTokenPair(refreshToken string) (accessToken, newRefreshToken string, err error) {
	// Validate refresh token
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return "", "", err
	}

	// Ensure it's a refresh token
	if err := s.validateTokenType(claims, "refresh"); err != nil {
		return "", "", err
	}

	// Generate new token pair
	return s.GenerateTokenPair(claims.Username)
}
