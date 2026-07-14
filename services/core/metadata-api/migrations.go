// Package metadataapi embeds the SQL migrations and applies them on boot.
// The migrations live next to this file under migrations/*.sql and are tracked
// in the schema_migrations table, so Migrate is idempotent.
package metadataapi

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

var versionRE = regexp.MustCompile(`^(\d+)_`)

type migrationFile struct {
	version string
	upSQL   string
}

func loadMigrations() ([]migrationFile, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	var ms []migrationFile
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		m := versionRE.FindStringSubmatch(name)
		if m == nil {
			return nil, fmt.Errorf("migration %q: filename must start with NNNN_", name)
		}
		b, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, err
		}
		ms = append(ms, migrationFile{version: m[1], upSQL: string(b)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].version < ms[j].version })
	return ms, nil
}

// rollback attempts to roll back a transaction; if that also fails it wraps
// the original error with the rollback failure.
func rollback(tx pgx.Tx, ctx context.Context, origErr error) error {
	if rbErr := tx.Rollback(ctx); rbErr != nil {
		return fmt.Errorf("%w (rollback: %v)", origErr, rbErr)
	}
	return origErr
}

// Migrate applies all pending migrations in order, each in its own transaction.
// It is safe to call on every boot.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	ms, err := loadMigrations()
	if err != nil {
		return err
	}
	for _, m := range ms {
		err := pool.QueryRow(ctx, "SELECT version FROM schema_migrations WHERE version=$1", m.version).Scan(new(string))
		if err == nil {
			continue // already applied
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("check migration %s: %w", m.version, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, m.upSQL); err != nil {
			return rollback(tx, ctx, fmt.Errorf("apply migration %s: %w", m.version, err))
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES ($1)", m.version); err != nil {
			return rollback(tx, ctx, fmt.Errorf("record migration %s: %w", m.version, err))
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}
