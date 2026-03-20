package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
// We build this once at startup and pass it around — no global variables
type Config struct {
	AppPort string
	AppEnv  string
}

// Load reads the .env file and returns a Config struct
// This is called once in main() before anything else
func Load() *Config {
	// Load .env file — if it fails (e.g. in production where env vars are set differently),
	// just log a warning and continue — don't crash
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	return &Config{
		// os.Getenv reads an environment variable
		// If APP_PORT isn't set, default to "8080"
		AppPort: getEnv("APP_PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),
	}
}

// getEnv is a helper that returns a default value if the env var isn't set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}