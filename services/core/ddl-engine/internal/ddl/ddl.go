// Package ddl turns metadata (entities/fields/indexes) into idempotent DDL and
// applies it to the tenant data DB. It is *additive*: it creates tables and adds
// columns/indexes that are missing, never drops or retypes existing ones. This
// is deliberately safe — the common case ("add an Asset entity → table appears")
// just works, and risky operations (type changes, drops) are deferred to a
// future explicit-migration path.
//
// Field types map to physical SQL types per METADATA.md. computed/formula fields
// are derived at the application layer and are NOT stored. Refs are soft (uuid)
// columns, matching the cross-database soft-reference model.
package ddl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/ddl-engine/internal/meta"
)

// Sentinel errors.
var (
	ErrNotFound    = errors.New("entity not found")
	ErrUnsupported = errors.New("unsupported field type")
)

// identRE bounds every identifier we emit. Lowercase-snake only; this also
// defeats SQL injection through table/column names and keeps names portable.
var identRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// Statement is one idempotent DDL operation in a Plan.
type Statement struct {
	Kind   string `json:"kind"`   // create_table | add_column | create_index
	Table  string `json:"table"`
	Detail string `json:"detail"` // column name, index name, …
	SQL    string `json:"sql"`
}

// Plan is the set of DDL statements needed to reconcile one entity.
type Plan struct {
	Entity     *meta.Entity `json:"entity"`
	Statements []Statement  `json:"statements"`
}

// ApplyResult is what running a Plan produced.
type ApplyResult struct {
	EntityID   string      `json:"entity_id"`
	Table      string      `json:"table"`
	Statements []Statement `json:"statements"`
	Applied    int         `json:"applied"`
}

// Migration is a recorded applied DDL statement.
type Migration struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	EntityID  string `json:"entity_id"`
	TableName string `json:"table_name"`
	Kind      string `json:"kind"`
	Detail    string `json:"detail"`
	Statement string `json:"statement"`
	AppliedAt string `json:"applied_at"`
}

// desiredIndex describes an index we want to exist.
type desiredIndex struct {
	name   string
	fields []string
	unique bool
}

// Engine reconciles metadata into the tenant data DB.
type Engine struct {
	meta *meta.Reader
	data *pgxpool.Pool
}

// New returns an Engine that reads from meta and writes DDL to data.
func New(reader *meta.Reader, data *pgxpool.Pool) *Engine {
	return &Engine{meta: reader, data: data}
}

// Meta returns the underlying metadata reader (used for boot reconcile).
func (e *Engine) Meta() *meta.Reader { return e.meta }

// Plan computes the additive DDL needed to reconcile one entity against the
// live tenant data DB. It performs no writes — callers can dry-run it.
func (e *Engine) Plan(ctx context.Context, entityID string) (*Plan, error) {
	if err := validateUUID(entityID); err != nil {
		return nil, err
	}
	ent, err := e.meta.EntityByID(ctx, entityID)
	if err != nil {
		return nil, mapMetaErr(err)
	}
	if err := validateIdent(ent.TableName); err != nil {
		return nil, fmt.Errorf("entity %q table_name %q: %w", ent.Name, ent.TableName, err)
	}
	fields, err := e.meta.Fields(ctx, entityID)
	if err != nil {
		return nil, err
	}
	indexes, err := e.meta.Indexes(ctx, entityID)
	if err != nil {
		return nil, err
	}

	exists, err := e.tableExists(ctx, ent.TableName)
	if err != nil {
		return nil, err
	}

	stmts := make([]Statement, 0, len(fields)+len(indexes)+2)
	if exists {
		// Table present: add any missing columns (nullable — required-ness is
		// enforced by data-api, so we never block an ALTER on existing rows).
		live, err := e.columns(ctx, ent.TableName)
		if err != nil {
			return nil, err
		}
		for _, f := range fields {
			s, skip, err := columnFor(f)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", f.Name, err)
			}
			if skip {
				continue
			}
			if err := validateIdent(f.Name); err != nil {
				return nil, fmt.Errorf("field %q: %w", f.Name, err)
			}
			if live[f.Name] {
				continue
			}
			stmts = append(stmts, Statement{
				Kind:   "add_column",
				Table:  ent.TableName,
				Detail: f.Name,
				SQL:    fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", quote(ent.TableName), quote(f.Name), s.sqlType),
			})
		}
	} else {
		// Brand-new table: emit one CREATE TABLE with every stored field inline
		// (NOT NULL where required) so it lands fully formed.
		cols := baseColumns()
		for _, f := range fields {
			s, skip, err := columnFor(f)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", f.Name, err)
			}
			if skip {
				continue
			}
			if err := validateIdent(f.Name); err != nil {
				return nil, fmt.Errorf("field %q: %w", f.Name, err)
			}
			line := fmt.Sprintf("%s %s", quote(f.Name), s.sqlType)
			if s.required {
				line += " NOT NULL"
			}
			cols = append(cols, line)
		}
		stmts = append(stmts, Statement{
			Kind:  "create_table",
			Table: ent.TableName,
			Detail: ent.TableName,
			SQL:   fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", quote(ent.TableName), strings.Join(cols, ",\n  ")),
		})
	}

	// Indexes (always additive): tenant_id is always indexed, plus metadata
	// indexes and config-driven per-field indexes.
	liveIdx, err := e.indexNames(ctx, ent.TableName)
	if err != nil {
		return nil, err
	}
	for _, ix := range desiredIndexes(ent.TableName, fields, indexes) {
		if liveIdx[ix.name] {
			continue
		}
		quoted := make([]string, len(ix.fields))
		for i, f := range ix.fields {
			quoted[i] = quote(f)
		}
		uniq := ""
		if ix.unique {
			uniq = "UNIQUE "
		}
		stmts = append(stmts, Statement{
			Kind:   "create_index",
			Table:  ent.TableName,
			Detail: ix.name,
			SQL:    fmt.Sprintf("CREATE %sINDEX IF NOT EXISTS %s ON %s (%s)", uniq, quote(ix.name), quote(ent.TableName), strings.Join(quoted, ", ")),
		})
	}

	return &Plan{Entity: ent, Statements: stmts}, nil
}

// Apply runs a Plan against the tenant data DB and records each statement.
func (e *Engine) Apply(ctx context.Context, entityID string) (*ApplyResult, error) {
	plan, err := e.Plan(ctx, entityID)
	if err != nil {
		return nil, err
	}
	res := &ApplyResult{EntityID: entityID, Table: plan.Entity.TableName, Statements: plan.Statements}
	for _, st := range plan.Statements {
		if _, err := e.data.Exec(ctx, st.SQL); err != nil {
			return res, fmt.Errorf("apply %q: %w", st.SQL, err)
		}
		if err := recordMigration(ctx, e.data, plan.Entity, st); err != nil {
			return res, fmt.Errorf("record migration: %w", err)
		}
		res.Applied++
	}
	return res, nil
}

// ApplyAll reconciles every entity owned by a tenant.
func (e *Engine) ApplyAll(ctx context.Context, tenantID string) ([]ApplyResult, error) {
	if err := validateUUID(tenantID); err != nil {
		return nil, err
	}
	entities, err := e.meta.EntitiesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]ApplyResult, 0, len(entities))
	for _, ent := range entities {
		res, err := e.Apply(ctx, ent.ID)
		if err != nil {
			return out, err
		}
		out = append(out, *res)
	}
	return out, nil
}

// Migrations lists recorded DDL migrations, optionally filtered.
func (e *Engine) Migrations(ctx context.Context, tenantID, entityID string) ([]Migration, error) {
	q := `SELECT id::text, tenant_id::text, entity_id::text, table_name, kind, detail, statement,
		applied_at::text FROM ddl_migrations`
	args := []any{}
	where := []string{}
	if tenantID != "" {
		args = append(args, tenantID)
		where = append(where, fmt.Sprintf("tenant_id = $%d::uuid", len(args)))
	}
	if entityID != "" {
		args = append(args, entityID)
		where = append(where, fmt.Sprintf("entity_id = $%d::uuid", len(args)))
	}
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY applied_at DESC LIMIT 500"
	rows, err := e.data.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Migration
	for rows.Next() {
		var m Migration
		if err := rows.Scan(&m.ID, &m.TenantID, &m.EntityID, &m.TableName, &m.Kind, &m.Detail, &m.Statement, &m.AppliedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ── live-schema probes ───────────────────────────────────────

func (e *Engine) tableExists(ctx context.Context, table string) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = $1)`
	var exists bool
	if err := e.data.QueryRow(ctx, q, table).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (e *Engine) columns(ctx context.Context, table string) (map[string]bool, error) {
	const q = `SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1`
	rows, err := e.data.Query(ctx, q, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

func (e *Engine) indexNames(ctx context.Context, table string) (map[string]bool, error) {
	const q = `SELECT indexname FROM pg_indexes
		WHERE schemaname = 'public' AND tablename = $1`
	rows, err := e.data.Query(ctx, q, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

func recordMigration(ctx context.Context, pool *pgxpool.Pool, ent *meta.Entity, st Statement) error {
	const q = `INSERT INTO ddl_migrations (tenant_id, entity_id, table_name, kind, detail, statement)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)`
	_, err := pool.Exec(ctx, q, ent.TenantID, ent.ID, ent.TableName, st.Kind, st.Detail, st.SQL)
	return err
}

// ── DDL generation helpers ───────────────────────────────────

type colSpec struct {
	sqlType  string
	required bool
}

// columnFor maps a metadata field to its physical column spec. ok=false means
// the field is not stored (computed/formula) and should be skipped.
func columnFor(f meta.Field) (colSpec, bool, error) {
	cfg := parseConfig(f.Config)
	switch f.Type {
	case "computed", "formula":
		return colSpec{}, true, nil // derived at app layer, never stored
	case "text", "longtext", "email", "url", "phone", "enum":
		return colSpec{sqlType: "text", required: cfg.Required}, false, nil
	case "richdoc", "multiselect", "multiref", "json", "file", "image":
		return colSpec{sqlType: "jsonb", required: cfg.Required}, false, nil
	case "number":
		return colSpec{sqlType: "bigint", required: cfg.Required}, false, nil
	case "decimal":
		return colSpec{sqlType: "numeric", required: cfg.Required}, false, nil
	case "money":
		return colSpec{sqlType: "numeric(19,4)", required: cfg.Required}, false, nil
	case "bool":
		return colSpec{sqlType: "boolean", required: cfg.Required}, false, nil
	case "date":
		return colSpec{sqlType: "date", required: cfg.Required}, false, nil
	case "datetime":
		return colSpec{sqlType: "timestamptz", required: cfg.Required}, false, nil
	case "time":
		return colSpec{sqlType: "time", required: cfg.Required}, false, nil
	case "ref", "agentref", "userref":
		return colSpec{sqlType: "uuid", required: cfg.Required}, false, nil
	default:
		return colSpec{}, false, fmt.Errorf("%w: %q", ErrUnsupported, f.Type)
	}
}

type fieldConfig struct {
	Required bool `json:"required"`
	Unique   bool `json:"unique"`
	Indexed  bool `json:"indexed"`
}

func parseConfig(raw json.RawMessage) fieldConfig {
	var c fieldConfig
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &c)
	}
	return c
}

// desiredIndexes assembles every index we want on a table:
//   - a tenant_id index (always, for scoping/RLS),
//   - one per indexed/unique field (from its config),
//   - one per declared `indexes` row.
// Later entries with the same name replace earlier ones (last wins).
func desiredIndexes(table string, fields []meta.Field, declared []meta.IndexDef) []desiredIndex {
	out := []desiredIndex{{name: indexName("idx", table, []string{"tenant_id"}), fields: []string{"tenant_id"}}}
	set := map[string]bool{}
	add := func(ix desiredIndex) {
		if !set[ix.name] {
			set[ix.name] = true
			out = append(out, ix)
		}
	}
	for _, f := range fields {
		if err := validateIdent(f.Name); err != nil {
			continue
		}
		cfg := parseConfig(f.Config)
		switch {
		case cfg.Unique:
			add(desiredIndex{name: indexName("udx", table, []string{f.Name}), fields: []string{f.Name}, unique: true})
		case cfg.Indexed:
			add(desiredIndex{name: indexName("idx", table, []string{f.Name}), fields: []string{f.Name}})
		}
	}
	for _, d := range declared {
		if len(d.Fields) == 0 {
			continue
		}
		ok := true
		for _, fn := range d.Fields {
			if validateIdent(fn) != nil {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		prefix := "idx"
		if d.Unique {
			prefix = "udx"
		}
		add(desiredIndex{name: indexName(prefix, table, d.Fields), fields: d.Fields, unique: d.Unique})
	}
	return out
}

// indexName builds a deterministic, length-safe index name.
func indexName(prefix, table string, fields []string) string {
	name := prefix + "_" + table + "_" + strings.Join(fields, "_")
	if len(name) <= 63 {
		return name
	}
	sum := sha256.Sum256([]byte(name))
	return name[:54] + hex.EncodeToString(sum[:4])
}

// baseColumns are the columns every generated table starts with.
func baseColumns() []string {
	return []string{
		`"id" uuid PRIMARY KEY DEFAULT gen_random_uuid()`,
		`"tenant_id" uuid NOT NULL`,
		`"created_at" timestamptz NOT NULL DEFAULT now()`,
		`"updated_at" timestamptz NOT NULL DEFAULT now()`,
		`"meta" jsonb NOT NULL DEFAULT '{}'::jsonb`,
	}
}

// ── identifier / value helpers ───────────────────────────────

func validateIdent(s string) error {
	if !identRE.MatchString(s) {
		return fmt.Errorf("invalid identifier %q (must match ^[a-z][a-z0-9_]{0,62}$)", s)
	}
	return nil
}

var uuidRE = regexp.MustCompile(`^(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validateUUID(s string) error {
	if !uuidRE.MatchString(s) {
		return fmt.Errorf("invalid id format %q", s)
	}
	return nil
}

func quote(id string) string {
	return `"` + strings.ReplaceAll(id, `"`, `""`) + `"`
}

func mapMetaErr(err error) error {
	if errors.Is(err, meta.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
