package wsgateway

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// AuthManager handles JWT authentication
type AuthManager struct {
	jwtSecret []byte
}

// NewAuthManager creates a new auth manager
func NewAuthManager(jwtSecret string) *AuthManager {
	return &AuthManager{
		jwtSecret: []byte(jwtSecret),
	}
}

// ValidateToken validates a JWT token and returns the user ID
func (a *AuthManager) ValidateToken(tokenString string) (string, error) {
	if a.jwtSecret == nil || len(a.jwtSecret) == 0 {
		// MVP: If no JWT secret is configured, allow all connections with default user
		// In production, this should be required
		return "default", nil
	}

	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	// Extract user ID
	userID, ok := claims["user_id"].(string)
	if !ok {
		// Try "sub" (subject) as fallback
		if sub, ok := claims["sub"].(string); ok {
			return sub, nil
		}
		return "", fmt.Errorf("user_id not found in token")
	}

	return userID, nil
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func (a *AuthManager) ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	// Support both "Bearer <token>" and just "<token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 {
		if strings.ToLower(parts[0]) != "bearer" {
			return "", fmt.Errorf("invalid authorization header format")
		}
		return parts[1], nil
	} else if len(parts) == 1 {
		// Allow just the token without "Bearer" prefix
		return parts[0], nil
	}

	return "", fmt.Errorf("invalid authorization header format")
}

