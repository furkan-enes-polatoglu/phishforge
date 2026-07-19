package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration, populated from environment variables.
type Config struct {
	// Core
	Mode        string // api | worker | migrate
	Environment string // dev | production

	// Database / cache
	DatabaseURL string
	RedisURL    string

	// HTTP
	AdminAddr    string // admin API + SPA
	PhishAddr    string // phishing / tracking server
	AdminBaseURL string
	PhishBaseURL string

	// Security
	JWTSecret   []byte
	RIDSecret   []byte // HMAC key for per-target tracking ids
	AccessTTL   time.Duration
	RefreshTTL  time.Duration
	CORSOrigins []string
	WebDist     string // path to built frontend (served if present)

	// Deliverability
	SpamdAddr string // host:port of a SpamAssassin spamd, empty = disabled

	// Mailgun webhook signing key (account-level secret, separate from any
	// per-profile API key) — verifies inbound delivered/bounced/complained
	// webhook authenticity. Empty disables signature verification (dev only).
	MailgunWebhookSigningKey string

	// Worker
	WorkerConcurrency int

	// Bootstrap admin (first run)
	BootstrapAdminUsername string
	BootstrapAdminPass     string
	BootstrapOrgName       string
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// Load builds a Config from the environment and validates required secrets.
func Load() (*Config, error) {
	c := &Config{
		Mode:                     getenv("PHISHFORGE_MODE", "api"),
		Environment:              getenv("PHISHFORGE_ENV", "dev"),
		DatabaseURL:              getenv("DATABASE_URL", "postgres://phishforge:phishforge@localhost:5432/phishforge?sslmode=disable"),
		RedisURL:                 getenv("REDIS_URL", "redis://localhost:6379/0"),
		AdminAddr:                getenv("ADMIN_ADDR", ":8080"),
		PhishAddr:                getenv("PHISH_ADDR", ":8081"),
		AdminBaseURL:             getenv("ADMIN_BASE_URL", "http://localhost:8080"),
		PhishBaseURL:             getenv("PHISH_BASE_URL", "http://localhost:8081"),
		JWTSecret:                []byte(getenv("JWT_SECRET", "")),
		RIDSecret:                []byte(getenv("RID_SECRET", "")),
		AccessTTL:                time.Duration(getenvInt("ACCESS_TTL_MIN", 30)) * time.Minute,
		RefreshTTL:               time.Duration(getenvInt("REFRESH_TTL_HOURS", 168)) * time.Hour,
		CORSOrigins:              splitList(getenv("CORS_ORIGINS", "http://localhost:5173,http://localhost:8080")),
		WebDist:                  getenv("WEB_DIST", "./web/dist"),
		SpamdAddr:                getenv("SPAMD_ADDR", ""),
		MailgunWebhookSigningKey: getenv("MAILGUN_WEBHOOK_SIGNING_KEY", ""),
		WorkerConcurrency:        getenvInt("WORKER_CONCURRENCY", 4),
		BootstrapAdminUsername:   getenv("BOOTSTRAP_ADMIN_USERNAME", ""),
		BootstrapAdminPass:       getenv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		BootstrapOrgName:         getenv("BOOTSTRAP_ORG_NAME", "Default Org"),
	}

	if len(c.JWTSecret) < 16 {
		return nil, fmt.Errorf("JWT_SECRET must be set (>=16 bytes); refusing to start with a weak/empty secret")
	}
	if len(c.RIDSecret) < 16 {
		return nil, fmt.Errorf("RID_SECRET must be set (>=16 bytes); refusing to start with a weak/empty secret")
	}
	return c, nil
}
