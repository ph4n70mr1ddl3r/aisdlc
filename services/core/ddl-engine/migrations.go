// Package ddlengine owns the schema bootstrapping ddl-engine performs on the
// tenant data DB: it creates the `ddl_migrations` ledger that records every
// applied DDL statement. Apply is idempotent and safe to call on every boot.
package ddlengine

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS ddl_migrations (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL,
    entity_id   uuid NOT NULL,
    table_name  text NOT NULL,
    kind        text NOT NULL,
    detail      text,
    statement   text NOT NULL,
    applied_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ddl_migrations_entity_idx ON ddl_migrations(entity_id);
CREATE INDEX IF NOT EXISTS ddl_migrations_tenant_idx ON ddl_migrations(tenant_id);
`

// EnsureSchema creates the ddl_migrations ledger if missing.
func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("ensure ddl_migrations: %w", err)
	}
	return nil
}
