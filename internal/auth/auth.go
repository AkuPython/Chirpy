package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)


func HashPassword(password string) (string, error) {
	pwd, err := bcrypt.GenerateFromPassword([]byte(password), 4)
	if err != nil {
		return "", err
	}
	return string(pwd), nil
}

func CheckPasswordHash(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer: "chirpy",
			IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
			Subject: userID.String(),
		},
	)

    // Sign the token with the secret key
    signedToken, err := token.SignedString([]byte(tokenSecret))
    if err != nil {
        return "", fmt.Errorf("failed to sign token: %w", err)
    }

    return signedToken, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
    token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(tokenSecret), nil
    })

    if err != nil {
        return uuid.Nil, fmt.Errorf("failed to parse token: %w", err)
    }

    if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
        // fmt.Println("Token is valid")
        // fmt.Printf("Issuer: %s\n", claims.Issuer)
        // fmt.Printf("Subject: %s\n", claims.Subject)
        // fmt.Printf("Issued At: %v\n", claims.IssuedAt)
        // fmt.Printf("Expires At: %v\n", claims.ExpiresAt)
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not convert subject to uuid %v ... %v", claims.Subject, err)
	}
	return userID, nil
    } else {
        fmt.Println("Token is invalid")
        return uuid.Nil, fmt.Errorf("token is invalid")
    }
}

func GetBearerToken(headers http.Header) (string, error) {
	auth_head := headers.Get("Authorization")
	if auth_head == "" {
		return "", fmt.Errorf("No Auth header")
	}
	auth_head = strings.TrimPrefix(auth_head, "Bearer")
	auth_head = strings.TrimSpace(auth_head)
	
	return auth_head, nil
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	//fmt.Printf("key_len: %v", key_int)
	hex_key := hex.EncodeToString(key)
	return hex_key, nil
}
