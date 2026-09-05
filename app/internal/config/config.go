package config

import (
	"fmt"
	"os"
	"strings"
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
	JSearchEnabled  bool
	// Multiple keys let quota be spread across more than one JSearch
	// account — JSearchSource rotates through them and falls over to the
	// next one on a quota/auth error instead of failing the whole fetch.
	JSearchAPIKeys []string
	JSearchCountry string
	GitHubUsername string
	GitHubToken    string
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
		JSearchEnabled:  envOrDefault("JSEARCH_ENABLED", "true") == "true",
		JSearchAPIKeys:  splitCommaList(os.Getenv("JSEARCH_API_KEYS")),
		JSearchCountry:  envOrDefault("JSEARCH_COUNTRY", "in"),
		GitHubUsername:  os.Getenv("GITHUB_USERNAME"),
		GitHubToken:     os.Getenv("GITHUB_TOKEN"),
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

// splitCommaList parses "a, b,c" into ["a", "b", "c"], dropping empty
// entries — used for JSEARCH_API_KEYS so more than one key can be supplied.
func splitCommaList(v string) []string {
	if v == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
