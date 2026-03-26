package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/unadkatdinky/devpulse/pkg/utils"
)

// contextKey is a custom type for context keys.
// We use a custom type instead of a plain string to avoid collisions —
// if two packages both store something in context with the key "userID",
// they'd overwrite each other. A custom type makes the key unique.
type contextKey string

const (
	// UserIDKey is the key we use to store/retrieve the user ID from context.
	UserIDKey contextKey = "userID"
	// UserEmailKey is the key for the user's email in context.
	UserEmailKey contextKey = "userEmail"
)

// RequireAuth is middleware that protects routes.
// It wraps any handler and checks for a valid JWT before letting it through.
// Usage: mux.HandleFunc("GET /protected", middleware.RequireAuth(handler, secret))
func RequireAuth(next http.HandlerFunc, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Get the Authorization header.
		// It should look like: "Bearer eyJhbGci..."
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.JSONError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		// Step 2: Check it starts with "Bearer " and extract the token.
		// strings.HasPrefix checks if the string starts with "Bearer "
		// strings.TrimPrefix removes "Bearer " leaving just the token string
		if !strings.HasPrefix(authHeader, "Bearer ") {
			utils.JSONError(w, http.StatusUnauthorized, "authorization header must start with Bearer")
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Step 3: Parse and validate the JWT.
		// jwt.ParseWithClaims decodes the token AND checks the signature.
		// jwt.MapClaims is a map — it holds the data baked into the token
		// (user ID, email, expiry time etc.)
		token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
			// This function is called to get the secret key for verification.
			// We return our JWT secret so the library can check the signature.
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			utils.JSONError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		// Step 4: Extract claims (the data inside the token).
		claims, ok := token.Claims.(*jwt.MapClaims)
		if !ok {
			utils.JSONError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		// Step 5: Put user info into the request context.
		// context.WithValue adds a key-value pair to the context.
		// We chain two calls — first add userID, then add userEmail.
		// The handler can then pull these out with r.Context().Value(key)
		ctx := context.WithValue(r.Context(), UserIDKey, (*claims)["sub"])
		ctx = context.WithValue(ctx, UserEmailKey, (*claims)["email"])

		// Step 6: Call the next handler with the enriched context.
		// r.WithContext returns a copy of the request with the new context.
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetUserID pulls the user ID out of the request context.
// Call this inside any protected handler to know who is making the request.
func GetUserID(r *http.Request) string {
	val := r.Context().Value(UserIDKey)
	if val == nil {
		return ""
	}
	// Type assert — we stored it as interface{}, get it back as string
	id, ok := val.(string)
	if !ok {
		return ""
	}
	return id
}

// GetUserEmail pulls the user email out of the request context.
func GetUserEmail(r *http.Request) string {
	val := r.Context().Value(UserEmailKey)
	if val == nil {
		return ""
	}
	email, ok := val.(string)
	if !ok {
		return ""
	}
	return email
}