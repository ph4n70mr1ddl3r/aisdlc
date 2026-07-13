// Package api wires the metadata-api HTTP layer: a generic CRUD router over the
// schema.Resource registry, plus tenant resolution and error mapping.
package api

import "github.com/ph4n70mr1ddl3r/aisdlc/services/core/metadata-api/internal/schema"

// Resources returns the full Layer 0–3 dictionary as generic resources.
// Add a row here to expose a new metadata table (its migration must exist).
func Resources() []*schema.Resource {
	return allResources
}

var (
	uuid0 = schema.TypeUUID
	text  = schema.TypeText
	intt  = schema.TypeInt
	boolT = schema.TypeBool
	jsonT = schema.TypeJSON
	ts    = schema.TypeTime

	idCols = []schema.Column{
		{Name: "id", Type: uuid0},
		{Name: "created_at", Type: ts},
		{Name: "updated_at", Type: ts},
	}
	idOnlyCol = []schema.Column{
		{Name: "id", Type: uuid0},
		{Name: "created_at", Type: ts},
	}
)

// helper to prepend standard columns to per-table settable columns.
func cols(extra ...schema.Column) []schema.Column {
	return append(append([]schema.Column{}, idCols...), extra...)
}
func colsNoUpdate(extra ...schema.Column) []schema.Column {
	return append(append([]schema.Column{}, idOnlyCol...), extra...)
}

var allResources = []*schema.Resource{
	// ── Layer 0: Tenancy & Identity ──────────────────────────
	{
		Name: "tenants", Singular: "tenant", Table: "tenants", IDColumn: "id",
		Columns: cols(
			{Name: "name", Type: text, Settable: true, Required: true},
			{Name: "slug", Type: text, Settable: true, Required: true},
			{Name: "plan", Type: text, Settable: true},
		),
		Searchable: []string{"name", "slug"}, OrderBy: "created_at DESC",
	},
	{
		Name: "users", Singular: "user", Table: "users", IDColumn: "id", TenantScope: true,
		Columns: cols(
			{Name: "tenant_id", Type: uuid0}, // injected; not settable
			{Name: "username", Type: text, Settable: true, Required: true},
			{Name: "password_hash", Type: text, Required: true}, // set server-side only
			{Name: "name", Type: text, Settable: true},
			{Name: "status", Type: text, Settable: true},
		),
		Searchable: []string{"username", "name"}, OrderBy: "created_at DESC",
	},
	{
		Name: "roles", Singular: "role", Table: "roles", IDColumn: "id", TenantScope: true,
		Columns: cols(
			{Name: "tenant_id", Type: uuid0},
			{Name: "name", Type: text, Settable: true, Required: true},
			{Name: "is_system", Type: boolT}, // system-managed only
		),
		Searchable: []string{"name"}, OrderBy: "created_at DESC",
	},
	{
		Name: "role_assignments", Singular: "role_assignment", Table: "role_assignments", IDColumn: "id",
		Columns: colsNoUpdate(
			{Name: "user_id", Type: uuid0, Settable: true, Required: true},
			{Name: "role_id", Type: uuid0, Settable: true, Required: true},
			{Name: "scope", Type: text, Settable: true},
		),
		OrderBy: "created_at DESC",
	},

	// ── Layer 1: Application Model ───────────────────────────
	{
		Name: "applications", Singular: "application", Table: "applications", IDColumn: "id", TenantScope: true,
		Columns: cols(
			{Name: "tenant_id", Type: uuid0},
			{Name: "name", Type: text, Settable: true, Required: true},
			{Name: "slug", Type: text, Settable: true, Required: true},
			{Name: "icon", Type: text, Settable: true},
			{Name: "version", Type: text, Settable: true},
			{Name: "status", Type: text, Settable: true},
			{Name: "description", Type: text, Settable: true},
		),
		Searchable: []string{"name", "slug"}, OrderBy: "created_at DESC",
	},
	{
		Name: "modules", Singular: "module", Table: "modules", IDColumn: "id",
		Columns: cols(
			{Name: "app_id", Type: uuid0, Settable: true, Required: true},
			{Name: "name", Type: text, Settable: true, Required: true},
			{Name: "order", Type: intt, Settable: true},
		),
		Searchable: []string{"name"}, OrderBy: "created_at DESC",
	},
	{
		Name: "app_dependencies", Singular: "app_dependency", Table: "app_dependencies", IDColumn: "id",
		Columns: colsNoUpdate(
			{Name: "app_id", Type: uuid0, Settable: true, Required: true},
			{Name: "depends_on_app_id", Type: uuid0, Settable: true, Required: true},
		),
		OrderBy: "created_at DESC",
	},

	// ── Layer 2: Data Model ──────────────────────────────────
	{
		Name: "entities", Singular: "entity", Table: "entities", IDColumn: "id",
		Columns: cols(
			{Name: "app_id", Type: uuid0, Settable: true, Required: true},
			{Name: "name", Type: text, Settable: true, Required: true},
			{Name: "table_name", Type: text, Settable: true, Required: true},
			{Name: "label_singular", Type: text, Settable: true},
			{Name: "label_plural", Type: text, Settable: true},
			{Name: "icon", Type: text, Settable: true},
			{Name: "is_audit", Type: boolT, Settable: true},
			{Name: "is_system", Type: boolT, Settable: true},
		),
		Searchable: []string{"name", "table_name"}, OrderBy: "created_at DESC",
	},
	{
		Name: "fields", Singular: "field", Table: "fields", IDColumn: "id",
		Columns: cols(
			{Name: "entity_id", Type: uuid0, Settable: true, Required: true},
			{Name: "name", Type: text, Settable: true, Required: true},
			{Name: "label", Type: text, Settable: true},
			{Name: "type", Type: text, Settable: true, Required: true},
			{Name: "config", Type: jsonT, Settable: true},
			{Name: "order", Type: intt, Settable: true},
			{Name: "is_system", Type: boolT, Settable: true},
		),
		Searchable: []string{"name", "label"}, OrderBy: "order ASC",
	},
	{
		Name: "relationships", Singular: "relationship", Table: "relationships", IDColumn: "id",
		Columns: cols(
			{Name: "name", Type: text, Settable: true, Required: true},
			{Name: "from_entity_id", Type: uuid0, Settable: true, Required: true},
			{Name: "to_entity_id", Type: uuid0, Settable: true, Required: true},
			{Name: "kind", Type: text, Settable: true, Required: true},
			{Name: "inverse_name", Type: text, Settable: true},
		),
		Searchable: []string{"name"}, OrderBy: "created_at DESC",
	},
	{
		Name: "indexes", Singular: "index", Table: "indexes", IDColumn: "id",
		Columns: cols(
			{Name: "entity_id", Type: uuid0, Settable: true, Required: true},
			{Name: "fields", Type: jsonT, Settable: true},
			{Name: "unique", Type: boolT, Settable: true},
		),
		OrderBy: "created_at DESC",
	},
	{
		Name: "validations", Singular: "validation", Table: "validations", IDColumn: "id",
		Columns: cols(
			{Name: "entity_id", Type: uuid0, Settable: true},
			{Name: "field_id", Type: uuid0, Settable: true},
			{Name: "expr", Type: text, Settable: true, Required: true},
			{Name: "message", Type: text, Settable: true},
		),
		OrderBy: "created_at DESC",
	},

	// ── Layer 3: UI Model ────────────────────────────────────
	{
		Name: "views", Singular: "view", Table: "views", IDColumn: "id",
		Columns: cols(
			{Name: "entity_id", Type: uuid0, Settable: true, Required: true},
			{Name: "type", Type: text, Settable: true, Required: true},
			{Name: "name", Type: text, Settable: true, Required: true},
			{Name: "config", Type: jsonT, Settable: true},
			{Name: "is_default", Type: boolT, Settable: true},
		),
		OrderBy: "created_at DESC",
	},
	{
		Name: "menus", Singular: "menu", Table: "menus", IDColumn: "id",
		Columns: cols(
			{Name: "app_id", Type: uuid0, Settable: true, Required: true},
			{Name: "parent_id", Type: uuid0, Settable: true},
			{Name: "label", Type: text, Settable: true, Required: true},
			{Name: "icon", Type: text, Settable: true},
			{Name: "target_type", Type: text, Settable: true, Required: true},
			{Name: "target_id", Type: uuid0, Settable: true},
			{Name: "order", Type: intt, Settable: true},
			{Name: "role_filter", Type: text, Settable: true},
		),
		OrderBy: "order ASC",
	},
}
