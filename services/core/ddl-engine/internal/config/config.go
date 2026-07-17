// Package config reads ddl-engine runtime settings from the environment.
package config

import (
	"fmt"
	"os"
)

// Config holds ddl-engine runtime configuration.
type Config struct {
	Addr            string // e.g. ":8000"
	MetaDB          string // metadata dictionary DB (read entities/fields/indexes)
	DataDB          string // tenant data DB (apply DDL here)
	ReconcileOnBoot bool   // if true, apply DDL for every entity on startup
}

// Load reads configuration from environment variables with sane defaults.
func Load() (Config, error) {
	meta := os.Getenv("META_DB")
	if meta == "" {
		return Config{}, fmt.Errorf("META_DB is required (metadata DB URL)")
	}
	data := os.Getenv("DATA_DB")
	if data == "" {
		return Config{}, fmt.Errorf("DATA_DB is required (tenant data DB URL)")
	}
	return Config{
		Addr:            ":" + envOr("PORT", "8000"),
		MetaDB:          meta,
		DataDB:          data,
		ReconcileOnBoot: envBool("RECONCILE_ON_BOOT"),
	}, nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envBool(k string) bool {
	switch os.Getenv(k) {
	case "1", "true", "TRUE", "yes":
		return true
	}
	return false
}
