package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Test password hashing & verification
func TestHashPassword(t *testing.T) {
	password := "securepassword"
	hashedPassword, err := HashPassword(password)

	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Hash should be valid
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		t.Errorf("Password hash verification failed: %v", err)
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "test123"
	hashedPassword, _ := HashPassword(password) // Ignore error for test simplicity

	// Correct password should not return error
	err := CheckPasswordHash(hashedPassword, password)
	if err != nil {
		t.Errorf("Expected password to match, but got error: %v", err)
	}

	// Wrong password should return error
	err = CheckPasswordHash(hashedPassword, "wrongpassword")
	if err == nil {
		t.Errorf("Expected password mismatch error, but got nil")
	}
}

// Test JWT Token Generation
func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	secret := "testsecret"
	expiresIn := time.Minute * 5

	token, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// Validate token format
	if token == "" {
		t.Errorf("Generated token is empty")
	}
}

// Test JWT Validation
func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "testsecret"
	expiresIn := time.Minute * 5

	token, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to create test token: %v", err)
	}

	// Validate the token
	validUserID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Errorf("Token validation failed: %v", err)
	}

	// Ensure user ID matches
	if validUserID != userID {
		t.Errorf("Expected user ID %v but got %v", userID, validUserID)
	}

	// Test with an invalid secret
	_, err = ValidateJWT(token, "wrongsecret")
	if err == nil {
		t.Errorf("Expected validation to fail with incorrect secret, but got nil")
	}
}
