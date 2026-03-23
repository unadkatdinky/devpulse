package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the data we put inside the JWT token
// This is what we can read back when a request comes in with a token
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
	// RegisteredClaims gives us standard fields:
	// ExpiresAt — when this token stops being valid
	// IssuedAt  — when this token was created
}

// GenerateToken creates a signed JWT for a user
// The token contains user_id and email so we know who's making requests
// expiryHours controls how long before the user must log in again
func GenerateToken(userID uint, email, secret string, expiryHours int) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(time.Duration(expiryHours) * time.Hour),
			),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	// NewWithClaims creates the header + payload parts
	// SignedString adds the signature using your secret key
	// Only your server knows the secret — only your server can make valid tokens
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken checks a token string is valid and returns its claims
// Called on every protected request — if this returns an error, reject the request
func ValidateToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// Verify the signing method is what we expect
			// Prevents algorithm confusion attacks where an attacker
			// sends a token signed with "none" as the algorithm
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}