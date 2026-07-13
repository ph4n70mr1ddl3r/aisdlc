// Package schema describes the metadata dictionary tables as declarative
// resources, so the generic store can do typed CRUD over all of Layers 0–3
// without per-table handlers. This is the Go encoding of METADATA.md.
package schema

// ColType enumerates value kinds the generic store knows how to coerce.
type ColType string

const (
	TypeText ColType = "text"
	TypeUUID ColType = "uuid"
	TypeInt  ColType = "int"
	TypeBool ColType = "bool"
	TypeJSON ColType = "json" // jsonb column
	TypeTime ColType = "ts"   // timestamptz
)

// Column describes one column of a resource.
type Column struct {
	Name      string  // DB column name
	Type      ColType
	Settable  bool // false for generated columns (id, *_at, tenant_id); ignored on write
	Required  bool // must be present and non-empty on create
	Sensitive bool // true for secrets (password_hash, tokens); excluded from GET responses
}

// Resource describes a dictionary table exposed as CRUD.
type Resource struct {
	Name        string   // plural URL segment, e.g. "entities"
	Singular    string   // singular label, for error messages
	Table       string   // DB table name
	IDColumn    string   // primary key (always "id" here)
	TenantScope bool     // if true, list/get/create are scoped by tenant_id from context
	Columns     []Column // all columns, in table order
	Searchable  []string // text columns scanned by ?q=
	OrderBy     string   // default ORDER BY, e.g. "created_at DESC"
}

// Column returns the column named n (by DB name), or nil if absent.
func (r *Resource) Column(n string) *Column {
	for i := range r.Columns {
		if r.Columns[i].Name == n {
			return &r.Columns[i]
		}
	}
	return nil
}

// Writable returns the settable columns (used by create/update).
func (r *Resource) Writable() []Column {
	out := make([]Column, 0, len(r.Columns))
	for _, c := range r.Columns {
		if c.Settable {
			out = append(out, c)
		}
	}
	return out
}
