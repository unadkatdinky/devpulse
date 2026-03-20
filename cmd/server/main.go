// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"
// )

// func main() {
// 	// Create a new ServeMux — this is Go's built-in router
// 	// Think of it like Express's app object: const app = express()
// 	mux := http.NewServeMux()

// 	// Register routes — same idea as app.get('/health', handler)
// 	// We're writing the handlers in the next step
// 	mux.HandleFunc("/health", healthHandler)

// 	// Tell the user the server is starting
// 	fmt.Println("🚀 DevPulse server running on http://localhost:8080")

// 	// Start listening for requests on port 8080
// 	// log.Fatal means: if this ever returns an error, log it and exit
// 	// ListenAndServe blocks forever — it's the infinite loop that keeps the server alive
// 	log.Fatal(http.ListenAndServe(":8080", mux))
// }

// // healthHandler is our first HTTP handler
// // Every handler in Go has EXACTLY this signature:
// // - w http.ResponseWriter → you write your response to this (like res in Express)
// // - r *http.Request       → the incoming request (like req in Express)
// func healthHandler(w http.ResponseWriter, r *http.Request) {
// 	// Set the Content-Type header so the browser/client knows we're sending JSON
// 	w.Header().Set("Content-Type", "application/json")

// 	// Write the response body
// 	// Fprintf writes formatted text to w (our response writer)
// 	fmt.Fprintf(w, `{"status": "ok", "service": "devpulse"}`)
// }

// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"

// 	// Import our handlers package using the module path from go.mod
// 	"github.com/unadkatdinky/devpulse/internal/handlers"
// )

// func main() {
// 	mux := http.NewServeMux()

// 	// Now we call the exported functions from the handlers package
// 	mux.HandleFunc("/health", handlers.HealthHandler)
// 	mux.HandleFunc("/api/events", handlers.EventsHandler)

// 	fmt.Println("🚀 DevPulse server running on http://localhost:8080")
// 	fmt.Println("   GET /health      → health check")
// 	fmt.Println("   GET /api/events  → list events (empty for now)")

// 	log.Fatal(http.ListenAndServe(":8080", mux))
// }

// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"

// 	"github.com/unadkatdinky/devpulse/internal/handlers"
// 	"github.com/unadkatdinky/devpulse/internal/middleware"
// )

// func main() {
// 	mux := http.NewServeMux()

// 	mux.HandleFunc("/health", handlers.HealthHandler)
// 	mux.HandleFunc("/api/events", handlers.EventsHandler)

// 	// Wrap the entire mux with our Logger middleware
// 	// Every request goes through Logger first, THEN to the mux, THEN to the handler
// 	// This is the middleware chain: Request → Logger → Mux → Handler → Logger → Response
// 	loggedMux := middleware.Logger(mux)

// 	fmt.Println("🚀 DevPulse server running on http://localhost:8080")

// 	log.Fatal(http.ListenAndServe(":8080", loggedMux))
// }

package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/unadkatdinky/devpulse/internal/config"
	"github.com/unadkatdinky/devpulse/internal/handlers"
	"github.com/unadkatdinky/devpulse/internal/middleware"
)

func main() {
	// Load config FIRST — before anything else
	cfg := config.Load()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/api/events", handlers.EventsHandler)

	loggedMux := middleware.Logger(mux)

	addr := ":" + cfg.AppPort
	fmt.Printf("🚀 DevPulse server running on http://localhost%s (env: %s)\n", addr, cfg.AppEnv)

	log.Fatal(http.ListenAndServe(addr, loggedMux))
} 