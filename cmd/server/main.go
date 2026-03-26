package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/unadkatdinky/devpulse/internal/config"
	"github.com/unadkatdinky/devpulse/internal/database"
	"github.com/unadkatdinky/devpulse/internal/handlers"
	"github.com/unadkatdinky/devpulse/internal/middleware"
	"github.com/unadkatdinky/devpulse/internal/repository"
	"github.com/unadkatdinky/devpulse/internal/worker"
)

func main() {
	// Load configuration from .env
	cfg := config.Load()

	// Connect to PostgreSQL
	db := database.Connect(cfg)

	// Run migrations — creates/updates all tables
	database.Migrate(db)

	// Start worker pool
	pool := worker.New(db, cfg.WorkerPoolSize)
	pool.Start()

	// Create repositories — these are the database layer
	userRepo := repository.NewUserRepository(db)
	eventRepo := repository.NewEventRepository(db)

	// Create handlers — these are the HTTP layer
	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret, 24)
	webhookHandler := handlers.NewWebhookHandler(db, pool, cfg.GitHubWebhookSecret)
	dashboardHandler := handlers.NewDashboardHandler(eventRepo)

	// Router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","version":"day4"}`)
	})

	// Auth routes — public, no JWT required
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)

	// Webhook route — public but HMAC protected
	mux.HandleFunc("POST /webhooks/github", webhookHandler.HandleGitHubWebhook)

	// Dashboard routes — protected by JWT middleware
	// middleware.RequireAuth wraps each handler — the JWT check runs first,
	// and only if it passes does the actual handler run
	mux.HandleFunc("GET /dashboard/stats",
		middleware.RequireAuth(dashboardHandler.GetStats, cfg.JWTSecret))

	mux.HandleFunc("GET /dashboard/events",
		middleware.RequireAuth(dashboardHandler.GetEvents, cfg.JWTSecret))

	mux.HandleFunc("GET /dashboard/events/{id}",
		middleware.RequireAuth(dashboardHandler.GetEventByID, cfg.JWTSecret))

	// Wrap everything with the logger middleware from Day 1
	handler := middleware.Logger(mux)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("DevPulse server running on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}