package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/unadkatdinky/devpulse/internal/config"
	"github.com/unadkatdinky/devpulse/internal/models"
)

// Connect opens a PostgreSQL connection and returns a *gorm.DB
// *gorm.DB is a connection pool — GORM manages multiple open connections
// internally so requests don't wait for each other
// Call this once in main() — pass the result to everything that needs DB access
func Connect(cfg *config.Config) *gorm.DB {
	// DSN = Data Source Name — the full address of your database
	// Locally:      host=localhost port=5432 user=postgres ...
	// On Railway:   you replace these values with what Railway gives you
	// Your Go code never changes — only the values in environment variables
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	// Pick the log level based on environment
	// Development: log every SQL query (great for learning what GORM generates)
	// Production: only log errors (too noisy otherwise)
	logLevel := logger.Info
	if cfg.AppEnv == "production" {
		logLevel = logger.Error
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})

	if err != nil {
		// Can't connect to DB = can't run. Log and exit immediately.
		log.Fatal("❌ Failed to connect to database: ", err)
	}

	// Get the underlying sql.DB to configure the connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("❌ Failed to get underlying DB: ", err)
	}

	// Maximum number of open connections to the database
	// Don't set this too high — PostgreSQL has a connection limit
	// Free tier on Railway/Render typically allows 20-25 connections
	sqlDB.SetMaxOpenConns(20)

	// Idle connections kept open and ready (faster than opening a new one)
	sqlDB.SetMaxIdleConns(10)

	log.Println("✅ Connected to PostgreSQL")
	return db
}

// Migrate creates or updates tables based on your model structs
// Safe to run every startup — only adds, never deletes
// In production you would use proper migration files (we'll add that on Day 7)
func Migrate(db *gorm.DB) {
	log.Println("Running migrations...")

	err := db.AutoMigrate(
    &models.User{},
    &models.GitHubEvent{}, // ← add this line
)

	if err != nil {
		log.Fatal("❌ Migration failed: ", err)
	}

	log.Println("✅ Migrations complete")
}