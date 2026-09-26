package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultPort        = "8080"
	defaultDatabaseURL = "postgres://agnos:agnos@localhost:5432/agnos?sslmode=disable"
	defaultJWTSecret   = "local-development-secret-change-me"
	defaultJWTExpiry   = time.Hour
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   time.Duration
}

func Load() Config {
	// A missing .env file is valid; deployed environments usually inject variables.
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = defaultJWTSecret
	}

	jwtExpiry := defaultJWTExpiry
	if configuredExpiry := os.Getenv("JWT_EXPIRY"); configuredExpiry != "" {
		if parsedExpiry, err := time.ParseDuration(configuredExpiry); err == nil && parsedExpiry > 0 {
			jwtExpiry = parsedExpiry
		}
	}

	return Config{
		Port:        port,
		DatabaseURL: databaseURL,
		JWTSecret:   jwtSecret,
		JWTExpiry:   jwtExpiry,
	}
}
