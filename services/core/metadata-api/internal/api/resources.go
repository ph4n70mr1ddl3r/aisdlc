package api

import (
	"errors"

	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/metadata-api/internal/schema"
)

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

func cols(extra ...schema.Column) []schema.Column {
	return append(append([]schema.Column{}, idCols...), extra...)
}

func colsNoUpdate(extra ...schema.Column) []schema.Column {
	return append(append([]schema.Column{}, idOnlyCol...), extra...)
}

var allResources = []*schema.Resource{
	{
		Name: "tenants", Singular: "tenant", Table: "tenants", IDColumn: "id",
		Columns: cols(
			schema.Column{Name: "name", Type: text, Settable: true, Required: true},
			schema.Column{Name: "slug", Type: text, Settable: true, Required: true},
			schema.Column{Name: "plan", Type: text, Settable: true},
		),
		Searchable: []string{"name", "slug"}, OrderBy: "created_at DESC",
	},
	{
		Name: "users", Singular: "user", Table: "users", IDColumn: "id", TenantScope: true,
		Columns: cols(
			schema.Column{Name: "tenant_id", Type: uuid0},
			schema.Column{Name: "username", Type: text, Settable: true, Required: true},
			schema.Column{Name: "password_hash", Type: text, Settable: true, Required: true, Sensitive: true},
			schema.Column{Name: "name", Type: text, Settable: true},
			schema.Column{Name: "status", Type: text, Settable: true},
		),
		Searchable: []string{"username", "name"}, OrderBy: "created_at DESC",
	},
	{
		Name: "roles", Singular: "role", Table: "roles", IDColumn: "id", TenantScope: true,
		Columns: cols(
			schema.Column{Name: "tenant_id", Type: uuid0},
			schema.Column{Name: "name", Type: text, Settable: true, Required: true},
			schema.Column{Name: "is_system", Type: boolT},
		),
		Searchable: []string{"name"}, OrderBy: "created_at DESC",
	},
	{
		Name: "role_assignments", Singular: "role_assignment", Table: "role_assignments", IDColumn: "id",
		Columns: colsNoUpdate(
			schema.Column{Name: "user_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "role_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "scope", Type: text, Settable: true},
		),
		OrderBy: "created_at DESC",
	},

	{
		Name: "applications", Singular: "application", Table: "applications", IDColumn: "id", TenantScope: true,
		Columns: cols(
			schema.Column{Name: "tenant_id", Type: uuid0},
			schema.Column{Name: "name", Type: text, Settable: true, Required: true},
			schema.Column{Name: "slug", Type: text, Settable: true, Required: true},
			schema.Column{Name: "icon", Type: text, Settable: true},
			schema.Column{Name: "version", Type: text, Settable: true},
			schema.Column{Name: "status", Type: text, Settable: true},
			schema.Column{Name: "description", Type: text, Settable: true},
		),
		Searchable: []string{"name", "slug"}, OrderBy: "created_at DESC",
	},
	{
		Name: "modules", Singular: "module", Table: "modules", IDColumn: "id",
		Columns: cols(
			schema.Column{Name: "app_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "name", Type: text, Settable: true, Required: true},
			schema.Column{Name: "order", Type: intt, Settable: true},
		),
		Searchable: []string{"name"}, OrderBy: "created_at DESC",
	},
	{
		Name: "app_dependencies", Singular: "app_dependency", Table: "app_dependencies", IDColumn: "id",
		Columns: colsNoUpdate(
			schema.Column{Name: "app_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "depends_on_app_id", Type: uuid0, Settable: true, Required: true},
		),
		OrderBy: "created_at DESC",
	},

	{
		Name: "entities", Singular: "entity", Table: "entities", IDColumn: "id",
		Columns: cols(
			schema.Column{Name: "app_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "name", Type: text, Settable: true, Required: true},
			schema.Column{Name: "table_name", Type: text, Settable: true, Required: true},
			schema.Column{Name: "label_singular", Type: text, Settable: true},
			schema.Column{Name: "label_plural", Type: text, Settable: true},
			schema.Column{Name: "icon", Type: text, Settable: true},
			schema.Column{Name: "is_audit", Type: boolT, Settable: true},
			schema.Column{Name: "is_system", Type: boolT, Settable: true},
		),
		Searchable: []string{"name", "table_name"}, OrderBy: "created_at DESC",
	},
	{
		Name: "fields", Singular: "field", Table: "fields", IDColumn: "id",
		Columns: cols(
			schema.Column{Name: "entity_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "name", Type: text, Settable: true, Required: true},
			schema.Column{Name: "label", Type: text, Settable: true},
			schema.Column{Name: "type", Type: text, Settable: true, Required: true},
			schema.Column{Name: "config", Type: jsonT, Settable: true},
			schema.Column{Name: "order", Type: intt, Settable: true},
			schema.Column{Name: "is_system", Type: boolT, Settable: true},
		),
		Searchable: []string{"name", "label"}, OrderBy: "order ASC",
	},
	{
		Name: "relationships", Singular: "relationship", Table: "relationships", IDColumn: "id",
		Columns: cols(
			schema.Column{Name: "name", Type: text, Settable: true, Required: true},
			schema.Column{Name: "from_entity_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "to_entity_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "kind", Type: text, Settable: true, Required: true},
			schema.Column{Name: "inverse_name", Type: text, Settable: true},
		),
		Searchable: []string{"name"}, OrderBy: "created_at DESC",
	},
	{
		Name: "indexes", Singular: "index", Table: "indexes", IDColumn: "id",
		Columns: cols(
			schema.Column{Name: "entity_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "fields", Type: jsonT, Settable: true},
			schema.Column{Name: "unique", Type: boolT, Settable: true},
		),
		OrderBy: "created_at DESC",
	},
	{
		Name: "validations", Singular: "validation", Table: "validations", IDColumn: "id",
		Columns: cols(
			schema.Column{Name: "entity_id", Type: uuid0, Settable: true},
			schema.Column{Name: "field_id", Type: uuid0, Settable: true},
			schema.Column{Name: "expr", Type: text, Settable: true, Required: true},
			schema.Column{Name: "message", Type: text, Settable: true},
		),
		OrderBy: "created_at DESC",
		Validate: func(body map[string]any) error {
			_, hasEntity := body["entity_id"]
			_, hasField := body["field_id"]
			if !hasEntity && !hasField {
				return errors.New("validation: at least one of entity_id or field_id is required")
			}
			return nil
		},
	},

	{
		Name: "views", Singular: "view", Table: "views", IDColumn: "id",
		Columns: cols(
			schema.Column{Name: "entity_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "type", Type: text, Settable: true, Required: true},
			schema.Column{Name: "name", Type: text, Settable: true, Required: true},
			schema.Column{Name: "config", Type: jsonT, Settable: true},
			schema.Column{Name: "is_default", Type: boolT, Settable: true},
		),
		OrderBy: "created_at DESC",
	},
	{
		Name: "menus", Singular: "menu", Table: "menus", IDColumn: "id",
		Columns: cols(
			schema.Column{Name: "app_id", Type: uuid0, Settable: true, Required: true},
			schema.Column{Name: "parent_id", Type: uuid0, Settable: true},
			schema.Column{Name: "label", Type: text, Settable: true, Required: true},
			schema.Column{Name: "icon", Type: text, Settable: true},
			schema.Column{Name: "target_type", Type: text, Settable: true, Required: true},
			schema.Column{Name: "target_id", Type: uuid0, Settable: true},
			schema.Column{Name: "order", Type: intt, Settable: true},
			schema.Column{Name: "role_filter", Type: text, Settable: true},
		),
		OrderBy: "order ASC",
	},
}
