// package config

// import (
// 	"log"
// 	"os"

// 	"github.com/joho/godotenv"
// )

// // Config holds all configuration for the application
// // We build this once at startup and pass it around — no global variables
// type Config struct {
// 	AppPort string
// 	AppEnv  string
// }

// // Load reads the .env file and returns a Config struct
// // This is called once in main() before anything else
// func Load() *Config {
// 	// Load .env file — if it fails (e.g. in production where env vars are set differently),
// 	// just log a warning and continue — don't crash
// 	if err := godotenv.Load(); err != nil {
// 		log.Println("No .env file found, reading from environment")
// 	}

// 	return &Config{
// 		// os.Getenv reads an environment variable
// 		// If APP_PORT isn't set, default to "8080"
// 		AppPort: getEnv("APP_PORT", "8080"),
// 		AppEnv:  getEnv("APP_ENV", "development"),
// 	}
// }

// // getEnv is a helper that returns a default value if the env var isn't set
// func getEnv(key, defaultValue string) string {
// 	if value := os.Getenv(key); value != "" {
// 		return value
// 	}
// 	return defaultValue
// }

// package config

// import (
// 	"log"
// 	"os"

// 	"github.com/joho/godotenv"
// )

// // Config holds every setting the app needs
// // Built once at startup from environment variables
// // Passed around to everything that needs it — no global variables
// type Config struct {
// 	// Server
// 	AppPort string
// 	AppEnv  string

// 	// Database
// 	DBHost     string
// 	DBPort     string
// 	DBUser     string
// 	DBPassword string
// 	DBName     string
// 	DBSSLMode  string

// 	// JWT
// 	JWTSecret      string
// 	JWTExpiryHours string
// }

// func Load() *Config {
// 	// Load .env file
// 	// In production (Railway, Render, Fly.io) env vars are injected directly
// 	// godotenv just reads them from a file for local development
// 	// If no .env file exists it logs a warning and continues — doesn't crash
// 	if err := godotenv.Load(); err != nil {
// 		log.Println("No .env file found, reading from environment variables")
// 	}

// 	return &Config{
// 		AppPort: getEnv("APP_PORT", "8080"),
// 		AppEnv:  getEnv("APP_ENV", "development"),

// 		DBHost:     getEnv("DB_HOST", "localhost"),
// 		DBPort:     getEnv("DB_PORT", "5432"),
// 		DBUser:     getEnv("DB_USER", "postgres"),
// 		DBPassword: getEnv("DB_PASSWORD", ""),
// 		DBName:     getEnv("DB_NAME", "devpulse"),
// 		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

// 		JWTSecret:      getEnv("JWT_SECRET", ""),
// 		JWTExpiryHours: getEnv("JWT_EXPIRY_HOURS", "24"),
// 	}
// }

// // getEnv returns the env var value or a default if not set
// func getEnv(key, defaultValue string) string {
// 	if value := os.Getenv(key); value != "" {
// 		return value
// 	}
// 	return defaultValue
// }

package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Server
	Port string

	// Auth
	JWTSecret string

	// GitHub
	GitHubWebhookSecret string

	// Worker
	WorkerPoolSize int

	AppEnv    string
	DBSSLMode string
}

func Load() *Config {
	// Load .env file — if it doesn't exist (e.g. on Railway), that's fine,
	// Railway injects env vars directly
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — reading from environment")
	}

	workerPoolSize, err := strconv.Atoi(getEnv("WORKER_POOL_SIZE", "5"))
	if err != nil {
		workerPoolSize = 5
	}

	return &Config{
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", ""),
		DBName:              getEnv("DB_NAME", "devpulse"),
		Port:                getEnv("PORT", "8080"),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		GitHubWebhookSecret: getEnv("GITHUB_WEBHOOK_SECRET", ""),
		WorkerPoolSize:      workerPoolSize,
		AppEnv:    getEnv("APP_ENV", "development"),
		DBSSLMode: getEnv("DB_SSL_MODE", "disable"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}