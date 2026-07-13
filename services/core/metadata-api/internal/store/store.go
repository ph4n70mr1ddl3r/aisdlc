// Package store is a generic CRUD store over the metadata dictionary. Given a
// schema.Resource it does list/get/create/update/delete with pagination,
// filtering, search, and tenant scoping — no per-table SQL.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/metadata-api/internal/schema"
)

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Sentinel errors. Handlers map these to HTTP statuses.
var (
	ErrNotFound    = errors.New("not found")
	ErrConflict    = errors.New("conflict")
	ErrFKViolation = errors.New("foreign-key violation")
	ErrValidation  = errors.New("validation error")
	ErrTenantReq   = errors.New("tenant id required")
)

// Store is the generic CRUD store.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Store backed by the given pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// ListQuery controls list pagination/filtering/search.
type ListQuery struct {
	TenantID string            // applied when resource.TenantScope is true
	Filters  map[string]string // exact-match filters by DB column name
	Search   string            // ?q= — OR'd ILIKE over resource.Searchable
	Order    string            // "col [ASC|DESC]"; defaults to resource.OrderBy
	Limit    int
	Offset   int
}

// List returns rows (JSON-ready maps) and the total count (pre-pagination).
func (s *Store) List(ctx context.Context, r *schema.Resource, q ListQuery) ([]map[string]any, int64, error) {
	// total count
	countSQL, countArgs, err := applyFilters(
		squirrel.Select("COUNT(*)").From(quote(r.Table)).PlaceholderFormat(squirrel.Dollar), r, q,
	).ToSql()
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := s.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, mapErr(err)
	}

	// rows
	b := squirrel.Select(columnList(r)...).From(quote(r.Table)).PlaceholderFormat(squirrel.Dollar)
	b = applyFilters(b, r, q)
	b = b.OrderBy(orderClause(q.Order, r))
	limit := q.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}
	b = b.Limit(uint64(limit)).Offset(uint64(offset))

	sql, args, err := b.ToSql()
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	defer rows.Close()
	out, err := scanRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func validUUID(s string) bool {
	return uuidRE.MatchString(s)
}

// Get returns one row by id (and tenant_id if scoped).
func (s *Store) Get(ctx context.Context, r *schema.Resource, id, tenantID string) (map[string]any, error) {
	if !validUUID(id) {
		return nil, fmt.Errorf("%w: invalid id format", ErrValidation)
	}
	if r.TenantScope && tenantID != "" && !validUUID(tenantID) {
		return nil, fmt.Errorf("%w: invalid tenant_id format", ErrValidation)
	}
	b := squirrel.Select(columnList(r)...).From(quote(r.Table)).PlaceholderFormat(squirrel.Dollar).
		Where(squirrel.Expr(quote(r.IDColumn)+" = ?::uuid", id))
	b = scopeTenant(b, r, tenantID)
	sql, args, err := b.ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return out[0], nil
}

// Create inserts a row from body (settable columns only) and returns it.
func (s *Store) Create(ctx context.Context, r *schema.Resource, body map[string]any, tenantID string) (map[string]any, error) {
	if r.TenantScope && tenantID == "" {
		return nil, ErrTenantReq
	}
	if r.TenantScope && tenantID != "" && !validUUID(tenantID) {
		return nil, fmt.Errorf("%w: invalid tenant_id format", ErrValidation)
	}
	for _, c := range r.Columns {
		if c.Required && c.Settable {
			if v, ok := body[c.Name]; !ok || isEmpty(v) {
				return nil, fmt.Errorf("%w: %s is required", ErrValidation, c.Name)
			}
		}
	}

	matched, args, err := writeColumns(r, body)
	if err != nil {
		return nil, err
	}
	cols := make([]string, 0, len(matched)+1)
	exprs := make([]string, 0, len(matched)+1)
	for i, c := range matched {
		cols = append(cols, quote(c.Name))
		exprs = append(exprs, bindExpr(i+1, c.Type))
	}
	if r.TenantScope {
		args = append(args, tenantID)
		cols = append(cols, quote("tenant_id"))
		exprs = append(exprs, bindExpr(len(args), schema.TypeUUID))
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("%w: no fields to insert", ErrValidation)
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		quote(r.Table), strings.Join(cols, ", "), strings.Join(exprs, ", "), strings.Join(columnList(r), ", "))
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("insert returned no row")
	}
	return out[0], nil
}

// Update patches a row by id with the settable columns present in body.
func (s *Store) Update(ctx context.Context, r *schema.Resource, id string, body map[string]any, tenantID string) (map[string]any, error) {
	if !validUUID(id) {
		return nil, fmt.Errorf("%w: invalid id format", ErrValidation)
	}
	if r.TenantScope && tenantID != "" && !validUUID(tenantID) {
		return nil, fmt.Errorf("%w: invalid tenant_id format", ErrValidation)
	}
	matched, args, err := writeColumns(r, body)
	if err != nil {
		return nil, err
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("%w: no fields to update", ErrValidation)
	}
	sets := make([]string, len(matched))
	for i, c := range matched {
		sets[i] = fmt.Sprintf("%s = %s", quote(c.Name), bindExpr(i+1, c.Type))
	}
	if r.Column("updated_at") != nil {
		sets = append(sets, "updated_at = now()")
	}
	args = append(args, id)
	where := fmt.Sprintf("%s = $%d::uuid", quote(r.IDColumn), len(args))
	if r.TenantScope && tenantID != "" {
		args = append(args, tenantID)
		where += fmt.Sprintf(" AND %s = $%d::uuid", quote("tenant_id"), len(args))
	}
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s RETURNING %s",
		quote(r.Table), strings.Join(sets, ", "), where, strings.Join(columnList(r), ", "))
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return out[0], nil
}

// Delete removes a row by id (and tenant_id if scoped).
func (s *Store) Delete(ctx context.Context, r *schema.Resource, id, tenantID string) error {
	if !validUUID(id) {
		return fmt.Errorf("%w: invalid id format", ErrValidation)
	}
	if r.TenantScope && tenantID != "" && !validUUID(tenantID) {
		return fmt.Errorf("%w: invalid tenant_id format", ErrValidation)
	}
	args := []any{id}
	where := fmt.Sprintf("%s = $1::uuid", quote(r.IDColumn))
	if r.TenantScope && tenantID != "" {
		args = append(args, tenantID)
		where += fmt.Sprintf(" AND %s = $2::uuid", quote("tenant_id"))
	}
	sql := fmt.Sprintf("DELETE FROM %s WHERE %s", quote(r.Table), where)
	tag, err := s.pool.Exec(ctx, sql, args...)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────

func applyFilters(b squirrel.SelectBuilder, r *schema.Resource, q ListQuery) squirrel.SelectBuilder {
	if r.TenantScope && q.TenantID != "" {
		b = b.Where(squirrel.Expr(quote("tenant_id")+" = ?::uuid", q.TenantID))
	}
	// deterministic filter order
	keys := make([]string, 0, len(q.Filters))
	for k := range q.Filters {
		if r.Column(k) != nil {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		b = b.Where(squirrel.Expr(quote(k)+"::text = ?", q.Filters[k]))
	}
	if q.Search != "" && len(r.Searchable) > 0 {
		pat := "%" + q.Search + "%"
		or := make(squirrel.Or, 0, len(r.Searchable))
		for _, c := range r.Searchable {
			or = append(or, squirrel.Expr(quote(c)+" ILIKE ?", pat))
		}
		b = b.Where(or)
	}
	return b
}

func scopeTenant(b squirrel.SelectBuilder, r *schema.Resource, tenantID string) squirrel.SelectBuilder {
	if r.TenantScope && tenantID != "" {
		return b.Where(squirrel.Expr(quote("tenant_id")+" = ?::uuid", tenantID))
	}
	return b
}

// writeColumns returns the settable columns present in body (in definition
// order) with their coerced values, ready for INSERT/UPDATE.
func writeColumns(r *schema.Resource, body map[string]any) ([]schema.Column, []any, error) {
	matched := make([]schema.Column, 0, len(body))
	args := make([]any, 0, len(body))
	for _, c := range r.Writable() {
		raw, present := body[c.Name]
		if !present {
			continue
		}
		v, err := coerce(raw, c.Type)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %s: %v", ErrValidation, c.Name, err)
		}
		matched = append(matched, c)
		args = append(args, v)
	}
	return matched, args, nil
}

// coerce converts an untyped JSON value (from gin) to a pgx-friendly value.
func coerce(raw any, t schema.ColType) (any, error) {
	if raw == nil {
		return nil, nil
	}
	switch t {
	case schema.TypeJSON:
		b, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		return string(b), nil // assignment cast text -> jsonb
	case schema.TypeInt:
		switch n := raw.(type) {
		case float64:
			return int64(n), nil
		case int:
			return int64(n), nil
		case int64:
			return n, nil
		case string:
			return strconv.ParseInt(n, 10, 64)
		default:
			return nil, fmt.Errorf("want int, got %T", raw)
		}
	case schema.TypeBool:
		switch b := raw.(type) {
		case bool:
			return b, nil
		case string:
			return strconv.ParseBool(b)
		default:
			return nil, fmt.Errorf("want bool, got %T", raw)
		}
	default: // text, uuid, ts -> string only
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("want string, got %T", raw)
		}
		return s, nil
	}
}

// scanRows materializes all rows into JSON-ready maps, decoding pgx values.
func scanRows(rows pgx.Rows) ([]map[string]any, error) {
	fields := rows.FieldDescriptions()
	out := make([]map[string]any, 0)
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		m := make(map[string]any, len(fields))
		for i, fd := range fields {
			m[fd.Name] = decode(vals[i])
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, mapErr(err)
	}
	return out, nil
}

// decode makes pgx values JSON-safe.
func decode(v any) any {
	switch t := v.(type) {
	case [16]byte: // pgx uuid -> canonical string
		return fmt.Sprintf("%x-%x-%x-%x-%x", t[0:4], t[4:6], t[6:8], t[8:10], t[10:16])
	case []byte: // jsonb/json raw bytes -> raw JSON (not base64)
		return json.RawMessage(t)
	case time.Time:
		return t.UTC().Format(time.RFC3339Nano)
	case nil:
		return nil
	default:
		return t
	}
}

func columnList(r *schema.Resource) []string {
	out := make([]string, 0, len(r.Columns))
	for _, c := range r.Columns {
		if c.Sensitive {
			continue
		}
		if c.Type == schema.TypeUUID {
			// cast to text so values come back as plain strings, never [16]byte
			out = append(out, quote(c.Name)+"::text AS "+quote(c.Name))
		} else {
			out = append(out, quote(c.Name))
		}
	}
	return out
}

func orderClause(order string, r *schema.Resource) string {
	if order == "" {
		return r.OrderBy
	}
	parts := strings.Fields(order)
	if len(parts) == 0 || r.Column(parts[0]) == nil {
		return r.OrderBy
	}
	dir := "ASC"
	if len(parts) > 1 && strings.EqualFold(parts[1], "DESC") {
		dir = "DESC"
	}
	return quote(parts[0]) + " " + dir
}

// bindExpr renders a positional placeholder with a column-type cast, so a
// value is accepted regardless of the OID the driver sends. (text→jsonb and
// text→uuid have no implicit assignment cast, so we cast explicitly.)
func bindExpr(n int, t schema.ColType) string {
	suffix := ""
	switch t {
	case schema.TypeUUID:
		suffix = "::uuid"
	case schema.TypeJSON:
		suffix = "::jsonb"
	case schema.TypeInt:
		suffix = "::bigint"
	case schema.TypeBool:
		suffix = "::boolean"
	}
	return "$" + strconv.Itoa(n) + suffix
}

func isEmpty(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(t) == ""
	default:
		return false
	}
}

// quote returns a safely-quoted SQL identifier (also handles reserved words
// like "order" and "unique").
func quote(id string) string {
	return `"` + strings.ReplaceAll(id, `"`, `""`) + `"`
}

// mapErr translates pg errors into the store's sentinel errors.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return ErrConflict
		case "23503": // foreign_key_violation
			return ErrFKViolation
		case "23502": // not_null_violation
			return fmt.Errorf("%w: %s", ErrValidation, pgErr.Message)
		case "23514": // check_violation
			return fmt.Errorf("%w: %s", ErrValidation, pgErr.Message)
		}
	}
	return err
}
