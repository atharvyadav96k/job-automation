package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	DatabaseURL     string
	RedisURL        string
	GeminiAPIKey    string
	ServerAddr      string
	BasicAuthUser   string
	BasicAuthPass   string
	ResumeDir       string
	FrontendOrigin  string
	ScrapeInterval  time.Duration
	RemotiveEnabled bool
}

func Load() (Config, error) {
	scrapeInterval, err := time.ParseDuration(envOrDefault("SCRAPE_INTERVAL", "4h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse SCRAPE_INTERVAL: %w", err)
	}

	cfg := Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RedisURL:        os.Getenv("REDIS_URL"),
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
		ServerAddr:      envOrDefault("SERVER_ADDR", ":8080"),
		BasicAuthUser:   os.Getenv("API_BASIC_AUTH_USER"),
		BasicAuthPass:   os.Getenv("API_BASIC_AUTH_PASS"),
		ResumeDir:       envOrDefault("RESUME_DIR", "data"),
		FrontendOrigin:  envOrDefault("FRONTEND_ORIGIN", "http://localhost:5173"),
		ScrapeInterval:  scrapeInterval,
		RemotiveEnabled: envOrDefault("REMOTIVE_ENABLED", "true") == "true",
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
