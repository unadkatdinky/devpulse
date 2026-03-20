package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthResponse defines the shape of our response
// The backtick tags (` `) tell the JSON encoder what key names to use
// Without tags: {"Status":"ok"} — capital S (Go exports with capitals)
// With tags:    {"status":"ok"} — lowercase (what APIs should return)
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

// HealthHandler handles GET /health
// Note: exported functions start with a capital letter (like public in JS)
// unexported functions start lowercase (like private in JS)
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200 — explicit is better than implicit

	response := HealthResponse{
		Status:    "ok",
		Service:   "devpulse",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// json.NewEncoder(w).Encode() converts the struct to JSON and writes to w
	// This is better than fmt.Fprintf for real JSON responses
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}