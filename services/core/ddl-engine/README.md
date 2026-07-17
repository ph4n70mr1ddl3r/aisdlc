# ddl-engine

The DDL reconciler of the model-driven core (**M1**). It reads entity/field/index
metadata from the dictionary and applies **idempotent, additive** DDL to the
tenant data DB so a new entity becomes a real, queryable table — the foundation
`data-api` and the portal renderer build on.

See [METADATA.md](../../../METADATA.md) for the data model and field types.

## What it does

Given an entity, it:

1. Reads `entities` + `fields` + `indexes` from the metadata DB.
2. Maps each field type to a physical SQL column (text→`text`, number→`bigint`,
   money→`numeric(19,4)`, ref→`uuid`, json→`jsonb`, …). `computed`/`formula`
   fields are derived at the app layer and are **not stored**.
3. Reconciles against the **live** tenant data DB:
   - **New table** → one `CREATE TABLE` with base columns (`id`, `tenant_id`,
     `created_at`, `updated_at`, `meta jsonb` overflow) + every stored field
     inline (`NOT NULL` where `required`).
   - **Existing table** → `ALTER TABLE … ADD COLUMN` (nullable) for any missing
     field. It never drops or retypes existing columns — those need a future
     explicit-migration path.
   - **Indexes** → `CREATE [UNIQUE] INDEX IF NOT EXISTS` for `tenant_id` (always),
     declared `indexes`, and per-field `config.indexed` / `config.unique`.
4. Records every applied statement in the `ddl_migrations` ledger.

Refs are soft `uuid` columns (matching the cross-database soft-reference model),
not physical FKs.

## Endpoints

```
GET  /healthz
GET  /v1/entities/{id}/ddl            dry-run: the statements that would apply
POST /v1/entities/{id}/apply          apply DDL for one entity
POST /v1/apply                        apply DDL for every entity in a tenant (X-Tenant-ID)
GET  /v1/migrations?entity_id=&tenant_id=   the applied-DDL ledger
```

## Config (env)

| Var | Default | Notes |
|---|---|---|
| `META_DB` | — | metadata DB URL (required) |
| `DATA_DB` | — | tenant data DB URL (required) |
| `PORT` | `8000` | listen port |
| `RECONCILE_ON_BOOT` | `0` | `1` applies DDL for *all* entities on startup |
| `CORS_ORIGIN` | `http://localhost:3000` | |

## Run via docker compose

```bash
make up                                   # infra (both Postgres)
make dev SVC=ddl-engine                   # build + run ddl-engine
# or: docker compose --profile app up ddl-engine postgres-meta postgres-data
```

## Smoke test (end-to-end against live Postgres)

```bash
# 1. create a tenant + app + entity + fields via metadata-api
curl -s -X POST localhost:8000/v1/tenants -H 'Content-Type: application/json' \
  -d '{"name":"Acme","slug":"acme"}'          # → { id: <TENANT_ID>, ... }
APP=$(curl -s -X POST localhost:8000/v1/applications \
  -H 'Content-Type: application/json' -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"name":"Assets","slug":"assets"}' | jq -r .id)
ENT=$(curl -s -X POST localhost:8000/v1/entities -H 'Content-Type: application/json' \
  -d "{\"app_id\":\"$APP\",\"name\":\"Asset\",\"table_name\":\"assets\"}" | jq -r .id)
curl -s -X POST localhost:8000/v1/fields -H 'Content-Type: application/json' \
  -d "{\"entity_id\":\"$ENT\",\"name\":\"serial\",\"type\":\"text\",\"config\":{\"required\":true,\"indexed\":true}}"

# 2. preview, then apply via ddl-engine (port 8001 in dev)
curl -s localhost:8001/v1/entities/$ENT/ddl
curl -s -X POST localhost:8001/v1/entities/$ENT/apply

# 3. the table now exists in the tenant data DB with the right columns
psql "$DATA_DB" -c '\d assets'
curl -s localhost:8001/v1/migrations
```

## Status (M1)

**Done:** entity→table reconciliation (create/add-column/add-index), field-type
→ SQL-type mapping, additive idempotency, `ddl_migrations` ledger, dry-run +
apply + tenant-apply + list APIs, boot reconcile (opt-in), identifier hardening.

**Deferred:** type changes / renames / drops (need an explicit migration +
diff history); publish-event subscription (pending metadata-api's
draft→publish lifecycle); per-field `default` expressions.
