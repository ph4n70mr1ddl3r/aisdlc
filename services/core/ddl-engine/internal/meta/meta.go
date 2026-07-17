// Package meta reads the metadata dictionary (entities, fields, indexes) from
// the metadata DB. ddl-engine uses it to compute the desired schema, then
// reconciles it against the live tenant data DB. It is a *read-only* client.
package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when no metadata row matches the query.
var ErrNotFound = errors.New("metadata not found")

// Entity is a decoded row of the metadata `entities` table, plus its owning
// tenant (resolved via applications.tenant_id).
type Entity struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	AppID         string `json:"app_id"`
	Name          string `json:"name"`
	TableName     string `json:"table_name"`
	LabelSingular string `json:"label_singular"`
	LabelPlural   string `json:"label_plural"`
	Icon          string `json:"icon"`
	IsAudit       bool   `json:"is_audit"`
	IsSystem      bool   `json:"is_system"`
}

// Field is a decoded row of the metadata `fields` table. Config is left raw so
// callers unmarshal only what they need.
type Field struct {
	ID       string          `json:"id"`
	EntityID string          `json:"entity_id"`
	Name     string          `json:"name"`
	Label    string          `json:"label"`
	Type     string          `json:"type"`
	Config   json.RawMessage `json:"config"`
	Order    int             `json:"order"`
	IsSystem bool            `json:"is_system"`
}

// IndexDef is a decoded row of the metadata `indexes` table.
type IndexDef struct {
	ID       string
	EntityID string
	Fields   []string
	Unique   bool
}

// Reader queries the metadata dictionary.
type Reader struct {
	pool *pgxpool.Pool
}

// New returns a Reader backed by the given metadata pool.
func New(pool *pgxpool.Pool) *Reader { return &Reader{pool: pool} }

// EntityByID returns one entity (with its tenant_id) by id.
func (r *Reader) EntityByID(ctx context.Context, id string) (*Entity, error) {
	const q = `SELECT e.id, a.tenant_id, e.app_id, e.name, e.table_name,
		COALESCE(e.label_singular, ''), COALESCE(e.label_plural, ''), COALESCE(e.icon, ''),
		e.is_audit, e.is_system
		FROM entities e JOIN applications a ON a.id = e.app_id
		WHERE e.id = $1::uuid`
	e := &Entity{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&e.ID, &e.TenantID, &e.AppID, &e.Name, &e.TableName,
		&e.LabelSingular, &e.LabelPlural, &e.Icon, &e.IsAudit, &e.IsSystem)
	if err != nil {
		return nil, mapErr(err)
	}
	return e, nil
}

// Fields returns the fields for an entity in display order.
func (r *Reader) Fields(ctx context.Context, entityID string) ([]Field, error) {
	const q = `SELECT id, entity_id, name, COALESCE(label, ''), type, config, "order", is_system
		FROM fields WHERE entity_id = $1::uuid ORDER BY "order" ASC, name ASC`
	rows, err := r.pool.Query(ctx, q, entityID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []Field
	for rows.Next() {
		var f Field
		if err := rows.Scan(&f.ID, &f.EntityID, &f.Name, &f.Label, &f.Type, &f.Config, &f.Order, &f.IsSystem); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Indexes returns declared indexes for an entity.
func (r *Reader) Indexes(ctx context.Context, entityID string) ([]IndexDef, error) {
	const q = `SELECT id, entity_id, fields, "unique" FROM indexes WHERE entity_id = $1::uuid`
	rows, err := r.pool.Query(ctx, q, entityID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []IndexDef
	for rows.Next() {
		var ix IndexDef
		var raw []byte
		if err := rows.Scan(&ix.ID, &ix.EntityID, &raw, &ix.Unique); err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &ix.Fields); err != nil {
				return nil, fmt.Errorf("decode indexes.fields for %s: %w", ix.ID, err)
			}
		}
		out = append(out, ix)
	}
	return out, rows.Err()
}

// EntitiesByTenant returns every entity owned by a tenant (via its applications).
func (r *Reader) EntitiesByTenant(ctx context.Context, tenantID string) ([]Entity, error) {
	const q = `SELECT e.id, a.tenant_id, e.app_id, e.name, e.table_name,
		COALESCE(e.label_singular, ''), COALESCE(e.label_plural, ''), COALESCE(e.icon, ''),
		e.is_audit, e.is_system
		FROM entities e JOIN applications a ON a.id = e.app_id
		WHERE a.tenant_id = $1::uuid ORDER BY e.created_at ASC`
	return scanEntities(r.pool.Query(ctx, q, tenantID))
}

// AllEntities returns every entity across all tenants (used by boot reconcile).
func (r *Reader) AllEntities(ctx context.Context) ([]Entity, error) {
	const q = `SELECT e.id, a.tenant_id, e.app_id, e.name, e.table_name,
		COALESCE(e.label_singular, ''), COALESCE(e.label_plural, ''), COALESCE(e.icon, ''),
		e.is_audit, e.is_system
		FROM entities e JOIN applications a ON a.id = e.app_id
		ORDER BY e.created_at ASC`
	return scanEntities(r.pool.Query(ctx, q))
}

func scanEntities(rows pgx.Rows, err error) ([]Entity, error) {
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var out []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.TenantID, &e.AppID, &e.Name, &e.TableName,
			&e.LabelSingular, &e.LabelPlural, &e.Icon, &e.IsAudit, &e.IsSystem); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
