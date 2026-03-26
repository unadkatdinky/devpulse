package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/unadkatdinky/devpulse/internal/middleware"
	"github.com/unadkatdinky/devpulse/internal/repository"
	"github.com/unadkatdinky/devpulse/pkg/utils"
	"gorm.io/gorm"
)

// DashboardHandler holds the event repository.
type DashboardHandler struct {
	eventRepo *repository.EventRepository
}

// NewDashboardHandler creates a DashboardHandler.
func NewDashboardHandler(eventRepo *repository.EventRepository) *DashboardHandler {
	return &DashboardHandler{eventRepo: eventRepo}
}

// GetStats handles GET /dashboard/stats
// Returns aggregate statistics about all events.
func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	// Pull the user ID from context — just to show we know who's asking.
	// In Day 6 we'll use this to filter stats per user.
	userID := middleware.GetUserID(r)
	_ = userID // underscore means "I know I'm not using this yet"

	stats, err := h.eventRepo.GetStats()
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "failed to fetch stats")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// GetEvents handles GET /dashboard/events
// Returns a paginated list of events.
// Query params: ?page=1&page_size=20
func (h *DashboardHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	// Read pagination query params from the URL.
	// r.URL.Query().Get("page") reads ?page=1 from the URL.
	// strconv.Atoi converts the string "1" to the integer 1.
	page := 1
	pageSize := 20

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	events, total, err := h.eventRepo.ListEvents(page, pageSize)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "failed to fetch events")
		return
	}

	// Build a response that includes pagination metadata.
	// The frontend needs total so it knows how many pages to show.
	response := map[string]interface{}{
		"events":    events,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		// Total pages = ceiling of total/pageSize
		// Integer math: (total + pageSize - 1) / pageSize
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetEventByID handles GET /dashboard/events/{id}
// Returns a single event with its full payload.
func (h *DashboardHandler) GetEventByID(w http.ResponseWriter, r *http.Request) {
	// r.PathValue("id") extracts the {id} part from the URL pattern.
	// This is Go 1.22's built-in path parameter extraction —
	// no third party router needed.
	id := r.PathValue("id")
	if id == "" {
		utils.JSONError(w, http.StatusBadRequest, "missing event id")
		return
	}

	// Basic UUID format check — just make sure it looks like a UUID
	// before hitting the database.
	if len(strings.TrimSpace(id)) < 10 {
		utils.JSONError(w, http.StatusBadRequest, "invalid event id format")
		return
	}

	event, err := h.eventRepo.GetEventByID(id)
	if err != nil {
		// Check if it's a "not found" error specifically.
		// errors.Is checks if the error matches a specific error type.
		// gorm.ErrRecordNotFound is what GORM returns when no row matches.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONError(w, http.StatusNotFound, "event not found")
			return
		}
		utils.JSONError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(event)
}