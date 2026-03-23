package utils

import (
	"encoding/json"
	"net/http"
)

// JSONSuccess writes a successful JSON response
// Every success response looks like: { "data": ... }
func JSONSuccess(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": data,
	})
}

// JSONError writes an error JSON response
// Every error response looks like: { "error": "message here" }
func JSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}