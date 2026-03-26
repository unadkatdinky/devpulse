package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/unadkatdinky/devpulse/internal/cache"
	"github.com/unadkatdinky/devpulse/internal/config"
	"github.com/unadkatdinky/devpulse/internal/database"
	"github.com/unadkatdinky/devpulse/internal/handlers"
	"github.com/unadkatdinky/devpulse/internal/middleware"
	"github.com/unadkatdinky/devpulse/internal/queue"
	"github.com/unadkatdinky/devpulse/internal/repository"
	"github.com/unadkatdinky/devpulse/internal/worker"
)

func main() {
	// Load config
	cfg := config.Load()

	// Connect to PostgreSQL
	db := database.Connect(cfg)
	database.Migrate(db)

	// Connect to Redis
	redisClient := cache.Connect(cfg)

	// Create Redis queue
	eventQueue := queue.New(redisClient)

	// Create worker processor
	processor := worker.New(db)

	// Create a context that cancels when the server shuts down.
	// This is passed to the Redis workers so they know when to stop.
	ctx, cancel := context.WithCancel(context.Background())

	// Start Redis queue workers
	eventQueue.StartWorkers(ctx, cfg.WorkerPoolSize, processor.ProcessJob)

	// Create repositories
	userRepo := repository.NewUserRepository(db)
	eventRepo := repository.NewEventRepository(db)

	// Create handlers
	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret, 24)
	webhookHandler := handlers.NewWebhookHandler(db, eventQueue, cfg.GitHubWebhookSecret)
	dashboardHandler := handlers.NewDashboardHandler(eventRepo)

	// Router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","version":"day5"}`)
	})

	// Auth routes — rate limited to 5 requests per minute per IP
	mux.HandleFunc("POST /auth/register",
		middleware.RateLimit(authHandler.Register, redisClient,
			cfg.RateLimitRequests, cfg.RateLimitWindowSeconds))

	mux.HandleFunc("POST /auth/login",
		middleware.RateLimit(authHandler.Login, redisClient,
			cfg.RateLimitRequests, cfg.RateLimitWindowSeconds))

	// Webhook route
	mux.HandleFunc("POST /webhooks/github", webhookHandler.HandleGitHubWebhook)

	// Dashboard routes — JWT protected
	mux.HandleFunc("GET /dashboard/stats",
		middleware.RequireAuth(dashboardHandler.GetStats, cfg.JWTSecret))

	mux.HandleFunc("GET /dashboard/events",
		middleware.RequireAuth(dashboardHandler.GetEvents, cfg.JWTSecret))

	mux.HandleFunc("GET /dashboard/events/{id}",
		middleware.RequireAuth(dashboardHandler.GetEventByID, cfg.JWTSecret))

	// Wrap with logger
	handler := middleware.Logger(mux)

	// Create the HTTP server as a variable — this is needed for graceful shutdown.
	// Previously we just called http.ListenAndServe directly.
	// Now we need a reference to the server so we can call server.Shutdown() later.
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: handler,
	}

	// ── Graceful Shutdown Setup ───────────────────────────────────────────────
	// signal.NotifyContext creates a context that is cancelled when the OS
	// sends SIGINT (Ctrl+C) or SIGTERM (what Railway sends when deploying).
	// This is how your program knows "it's time to stop".
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine so it doesn't block.
	// This lets the main goroutine wait for the shutdown signal below.
	go func() {
		log.Printf("DevPulse server running on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Block here until Ctrl+C or SIGTERM is received.
	<-quit
	log.Println("Shutdown signal received — shutting down gracefully...")

	// Cancel the worker context — tells Redis workers to stop.
	cancel()

	// Give in-flight HTTP requests up to 10 seconds to finish.
	// After 10 seconds, force shutdown.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("Error closing Redis: %v", err)
	}

	log.Println("✅ Server exited cleanly")
}