package api

// Authentication service endpoints
const (
	// Service name
	AuthService = "auth.Auth"

	// Authentication endpoints
	AuthRegister      = "/auth.Auth/Register"
	AuthLogin         = "/auth.Auth/Login"
	AuthValidateToken = "/auth.Auth/ValidateToken"
	AuthRefreshToken  = "/auth.Auth/RefreshToken"
)

// PublicEndpoints defines endpoints that don't require authentication
var PublicEndpoints = map[string]bool{
	AuthRegister:      true,
	AuthLogin:         true,
	AuthValidateToken: true,
	AuthRefreshToken:  true,
}
