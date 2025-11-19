package wsgateway

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthManager_ValidateToken(t *testing.T) {
	secret := "test-secret-key"
	authManager := NewAuthManager(secret)

	// Create a valid token
	claims := jwt.MapClaims{
		"user_id": "user-1",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Validate token
	userID, err := authManager.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("Expected user ID %s, got %s", "user-1", userID)
	}
}

func TestAuthManager_ValidateToken_InvalidSecret(t *testing.T) {
	secret := "test-secret-key"
	authManager := NewAuthManager(secret)

	// Create token with different secret
	claims := jwt.MapClaims{
		"user_id": "user-1",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("wrong-secret"))
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Validate should fail
	_, err = authManager.ValidateToken(tokenString)
	if err == nil {
		t.Error("Expected error for token with wrong secret")
	}
}

func TestAuthManager_ValidateToken_NoSecret(t *testing.T) {
	// MVP: No secret should allow default user
	authManager := NewAuthManager("")

	userID, err := authManager.ValidateToken("any-token")
	if err != nil {
		t.Fatalf("Expected no error for MVP (no secret), got %v", err)
	}
	if userID != "default" {
		t.Errorf("Expected default user ID, got %s", userID)
	}
}

func TestAuthManager_ExtractTokenFromHeader(t *testing.T) {
	authManager := NewAuthManager("test-secret")

	// Test Bearer token
	token, err := authManager.ExtractTokenFromHeader("Bearer test-token")
	if err != nil {
		t.Fatalf("Failed to extract token: %v", err)
	}
	if token != "test-token" {
		t.Errorf("Expected token %s, got %s", "test-token", token)
	}

	// Test token without Bearer prefix
	token, err = authManager.ExtractTokenFromHeader("test-token")
	if err != nil {
		t.Fatalf("Failed to extract token: %v", err)
	}
	if token != "test-token" {
		t.Errorf("Expected token %s, got %s", "test-token", token)
	}

	// Test empty header
	_, err = authManager.ExtractTokenFromHeader("")
	if err == nil {
		t.Error("Expected error for empty header")
	}
}

func TestAuthManager_ValidateToken_SubjectClaim(t *testing.T) {
	secret := "test-secret-key"
	authManager := NewAuthManager(secret)

	// Create token with "sub" claim instead of "user_id"
	claims := jwt.MapClaims{
		"sub": "user-2",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Validate token (should use "sub" as fallback)
	userID, err := authManager.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}
	if userID != "user-2" {
		t.Errorf("Expected user ID %s, got %s", "user-2", userID)
	}
}

