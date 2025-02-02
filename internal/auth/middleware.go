package auth

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/elskow/chef-infra/internal/config"
)

// Define a custom type for context keys
type contextKey string

const (
	// UserContextKey is the key used to store the username in the context
	UserContextKey contextKey = "user"
)

type AuthMiddleware struct {
	config *config.AuthConfig
}

func NewAuthMiddleware(config *config.AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{
		config: config,
	}
}

func (m *AuthMiddleware) AuthenticationMiddleware(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	token := values[0] // Get the first token

	claims, err := validateToken(token, m.config.JWTSecret)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	// Use the custom context key type
	return context.WithValue(ctx, UserContextKey, claims.Username), nil
}

// Helper function to get username from context
func GetUserFromContext(ctx context.Context) (string, error) {
	username, ok := ctx.Value(UserContextKey).(string)
	if !ok {
		return "", errors.New("user not found in context")
	}
	return username, nil
}

func validateToken(tokenString string, secretKey string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
