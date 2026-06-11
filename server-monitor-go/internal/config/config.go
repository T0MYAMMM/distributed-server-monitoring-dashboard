// Package config centralizes runtime configuration for the backend server,
// sourced from environment variables with sensible defaults.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds all tunable backend settings.
type Config struct {
	// Addr is the host:port the HTTP server listens on. Binding to 0.0.0.0
	// makes the dashboard reachable on the host's Tailscale IP.
	Addr string
	// DatabasePath is the SQLite file location.
	DatabasePath string
	// SecretKey signs JWT auth tokens. Persisted so tokens survive restarts.
	SecretKey []byte
	// StaleAfterSeconds is how long a server may go silent before it is
	// marked "stopped" by the monitor.
	StaleAfterSeconds int
	// AgentsDir holds prebuilt agent binaries served at /download/<file> so
	// tailnet servers can self-install with a single command.
	AgentsDir string
}

// Load builds a Config from the environment, generating and persisting a JWT
// secret on first run when SECRET_KEY is not provided.
func Load() (*Config, error) {
	dataDir := getenv("DATA_DIR", "./data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}

	cfg := &Config{
		Addr:              getenv("ADDR", "0.0.0.0:5000"),
		DatabasePath:      getenv("DATABASE_PATH", filepath.Join(dataDir, "servers.db")),
		StaleAfterSeconds: getenvInt("STALE_AFTER_SECONDS", 30),
		AgentsDir:         getenv("AGENTS_DIR", "./dist"),
	}

	secret, err := loadOrCreateSecret(dataDir)
	if err != nil {
		return nil, err
	}
	cfg.SecretKey = secret
	return cfg, nil
}

// loadOrCreateSecret returns the JWT signing key: the SECRET_KEY env var if
// set, otherwise a random key persisted to <dataDir>/.secret so issued tokens
// remain valid across restarts.
func loadOrCreateSecret(dataDir string) ([]byte, error) {
	if v := os.Getenv("SECRET_KEY"); v != "" {
		return []byte(v), nil
	}

	secretPath := filepath.Join(dataDir, ".secret")
	if b, err := os.ReadFile(secretPath); err == nil && len(b) > 0 {
		return b, nil
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	secret := []byte(hex.EncodeToString(buf))
	if err := os.WriteFile(secretPath, secret, 0o600); err != nil {
		return nil, err
	}
	return secret, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getenvInt reads an integer env var, falling back on empty or invalid values
// so existing deployments that never set it run unchanged.
func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}
