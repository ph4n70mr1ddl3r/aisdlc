# metadata-api

The dictionary CRUD service — the foundation of the model-driven core (**M1**).
Typed create/read/update/delete over the metadata tables for Layers 0–3
(tenants, users, roles, applications, modules, entities, fields, relationships,
indexes, validations, views, menus). A single generic engine serves every
resource — add a `schema.Resource` + a migration to expose a new table.

See [METADATA.md](../../../METADATA.md) for the data model.

## Endpoints

```
GET    /healthz
GET    /v1/{resource}?limit=&offset=&order=&q=&tenant_id=&<col>=<val>
POST   /v1/{resource}
GET    /v1/{resource}/{id}
PATCH  /v1/{resource}/{id}
DELETE /v1/{resource}/{id}
```

`{resource}` ∈ `tenants, users, roles, role_assignments, applications, modules,
app_dependencies, entities, fields, relationships, indexes, validations, views,
menus`.

Tenant-scoped resources (`users`, `roles`, `applications`) require an
`X-Tenant-ID` header (or `?tenant_id=`) on create; lists are filtered by it when
present. Any query param that matches a column is an exact filter, e.g.
`/v1/fields?entity_id=<uuid>`.

## Config (env)

| Var | Default | Notes |
|---|---|---|
| `META_DB` | — | Postgres URL (required; `DATABASE_URL` fallback) |
| `PORT` | `8000` | listen port |

## Run via docker compose

The service is in the `app` profile and runs embedded migrations on boot:

```bash
make up                              # infra only (starts postgres-meta)
make dev SVC=metadata-api            # build + run metadata-api against it
# or: docker compose --profile app up metadata-api postgres-meta
```

## Build / run locally (needs Go 1.22+)

```bash
cd services/core/metadata-api
go mod tidy                         # fetch deps + write go.sum
go run ./cmd/server                 # needs META_DB pointing at a Postgres

# smoke test:
curl -s localhost:8000/healthz
curl -s -X POST localhost:8000/v1/tenants -H 'Content-Type: application/json' \
  -d '{"name":"Acme","slug":"acme"}'
curl -s localhost:8000/v1/tenants
```

## Migrations

`migrations/NNNN_*.up.sql` are embedded (`//go:embed`) and applied in order on
boot, each in its own transaction, tracked in `schema_migrations`. Validated
against PostgreSQL 16 (up → 15 tables, jsonb round-trip, down → 0, re-apply OK).

## Status (M1)

**Done:** schema + migrations (Layers 0–3); generic CRUD with validation,
tenant scoping, pagination, filtering, search; conflict / FK / validation error
mapping; embedded migrations; healthcheck.

**Next (rest of M1):** draft→publish lifecycle + versioned `metadata_bundles`;
publish events to NATS (`metadata.published`); then `ddl-engine` consumes this
metadata to create tenant tables. Auth lands in M2 (`identity`).
