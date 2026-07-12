// Package config reads metadata-api runtime settings from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds metadata-api runtime configuration.
type Config struct {
	Addr        string // e.g. ":8000"
	DatabaseURL string // postgres://...
}

// Load reads configuration from environment variables with sane defaults.
func Load() (Config, error) {
	db := os.Getenv("META_DB")
	if db == "" {
		db = os.Getenv("DATABASE_URL")
	}
	if db == "" {
		return Config{}, fmt.Errorf("META_DB (or DATABASE_URL) is required")
	}
	return Config{Addr: ":" + envOr("PORT", "8000"), DatabaseURL: db}, nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// helper kept for future boolean toggles (e.g. DEBUG).
func envBool(k string, d bool) bool {
	if v := os.Getenv(k); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return d
}
