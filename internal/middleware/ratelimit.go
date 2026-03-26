package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unadkatdinky/devpulse/pkg/utils"
)

// RateLimit returns a middleware that limits requests per IP address.
// maxRequests: how many requests allowed per window
// windowSeconds: the time window in seconds
func RateLimit(next http.HandlerFunc, redisClient *redis.Client, maxRequests int, windowSeconds int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the client's IP address.
		// r.RemoteAddr is "IP:port" — we just want the IP.
		ip := r.RemoteAddr
		// If behind a proxy (like Railway), use the real IP from the header
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			ip = realIP
		} else if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			ip = forwardedFor
		}

		// Build a unique Redis key for this IP + endpoint combination.
		// Example: "ratelimit:127.0.0.1:1:POST /auth/login"
		key := fmt.Sprintf("ratelimit:%s:%s", ip, r.URL.Path)

		ctx := context.Background()

		// Increment the counter for this key.
		// INCR adds 1 and returns the new value.
		// If the key doesn't exist, Redis creates it starting at 0, then adds 1.
		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			// If Redis is down, let the request through — don't block users
			// just because your cache is unavailable.
			log.Printf("RateLimit: Redis error: %v — allowing request", err)
			next.ServeHTTP(w, r)
			return
		}

		// On the first request, set the expiry.
		// We only set it on count==1 to avoid resetting the window on every request.
		if count == 1 {
			redisClient.Expire(ctx, key, time.Duration(windowSeconds)*time.Second)
		}

		// Add rate limit headers so the client knows their status.
		// This is standard practice — APIs like GitHub do the same.
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, int64(maxRequests)-count)))

		// If over the limit, reject the request.
		if count > int64(maxRequests) {
			utils.JSONError(w, http.StatusTooManyRequests, "too many requests — please wait a minute")
			return
		}

		next.ServeHTTP(w, r)
	}
}

// max returns the larger of two int64 values.
// Go 1.21+ has a built-in max() but we define it here for clarity.
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}