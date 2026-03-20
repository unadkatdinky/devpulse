package handlers

import (
	"encoding/json"
	"net/http"
)

// EventsHandler handles GET /api/events
// Returns an empty array for now — we'll fill this on Day 2 with real DB data
func EventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Empty slice for now — will be populated from DB on Day 2
	// Note: json.Marshal(nil) gives "null", json.Marshal([]string{}) gives "[]"
	// We want [] not null, so we use an empty slice
	events := []map[string]interface{}{}

	json.NewEncoder(w).Encode(events)
}