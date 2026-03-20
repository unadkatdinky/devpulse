package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logger wraps an http.Handler and logs every request
// This is the middleware pattern — it intercepts the request before it reaches the handler
func Logger(next http.Handler) http.Handler {
	// http.HandlerFunc converts our function into an http.Handler interface
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record when the request started
		start := time.Now()

		// Call the NEXT handler in the chain
		// This is where the actual work happens (HealthHandler, EventsHandler, etc.)
		next.ServeHTTP(w, r)

		// After the handler returns, log the details
		// time.Since(start) gives us the duration
		log.Printf(
			"%s %s %s",
			r.Method,          // GET, POST, etc.
			r.URL.Path,        // /health, /api/events, etc.
			time.Since(start), // how long it took: 245µs
		)
	})
}