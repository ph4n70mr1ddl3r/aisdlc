-- 0001_init.up.sql — metadata dictionary, Layers 0–3 (METADATA.md)
-- Applied by metadata-api on boot (internal/store/migrations.go).
-- Target: PostgreSQL 13+ (gen_random_uuid is built-in).

BEGIN;

-- ── Layer 0: Tenancy & Identity ──────────────────────────────
CREATE TABLE tenants (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    slug        text NOT NULL UNIQUE,
    plan        text NOT NULL DEFAULT 'standard',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username       text NOT NULL,
    password_hash  text NOT NULL,
    name           text,
    status         text NOT NULL DEFAULT 'active',
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, username)
);

-- sessions is internal to identity (M2); created here, not exposed via CRUD.
CREATE TABLE sessions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       text NOT NULL UNIQUE,
    expires     timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE roles (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        text NOT NULL,
    is_system   boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE role_assignments (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scope       text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, role_id, scope)
);

-- ── Layer 1: Application Model ───────────────────────────────
CREATE TABLE applications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        text NOT NULL,
    slug        text NOT NULL,
    icon        text,
    version     text NOT NULL DEFAULT '0.1.0',
    status      text NOT NULL DEFAULT 'draft',   -- draft | published
    description text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, slug)
);

CREATE TABLE modules (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id      uuid NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name        text NOT NULL,
    "order"     integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE app_dependencies (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id             uuid NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    depends_on_app_id  uuid NOT NULL REFERENCES applications(id) ON DELETE RESTRICT,
    created_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (app_id, depends_on_app_id)
);

-- ── Layer 2: Data Model ──────────────────────────────────────
CREATE TABLE entities (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          uuid NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name            text NOT NULL,
    table_name      text NOT NULL,
    label_singular  text,
    label_plural    text,
    icon            text,
    is_audit        boolean NOT NULL DEFAULT false,
    is_system       boolean NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (app_id, name),
    UNIQUE (table_name)
);

CREATE TABLE fields (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id   uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    name        text NOT NULL,
    label       text,
    type        text NOT NULL,
    config      jsonb NOT NULL DEFAULT '{}'::jsonb,
    "order"     integer NOT NULL DEFAULT 0,
    is_system   boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (entity_id, name)
);

CREATE TABLE relationships (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name            text NOT NULL,
    from_entity_id  uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    to_entity_id    uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    kind            text NOT NULL,    -- oneToMany | manyToOne | manyToMany
    inverse_name    text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (from_entity_id, to_entity_id, name)
);

CREATE TABLE indexes (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id   uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    fields      jsonb NOT NULL DEFAULT '[]'::jsonb,
    "unique"    boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE validations (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id   uuid REFERENCES entities(id) ON DELETE CASCADE,
    field_id    uuid REFERENCES fields(id) ON DELETE CASCADE,
    expr        text NOT NULL,
    message     text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (entity_id IS NOT NULL OR field_id IS NOT NULL)
);

-- ── Layer 3: UI Model ────────────────────────────────────────
CREATE TABLE views (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id   uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    type        text NOT NULL,    -- list|form|detail|kanban|calendar|gallery|dashboard
    name        text NOT NULL,
    config      jsonb NOT NULL DEFAULT '{}'::jsonb,
    is_default  boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (entity_id, type, name)
);

CREATE TABLE menus (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id       uuid NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    parent_id    uuid REFERENCES menus(id) ON DELETE CASCADE,
    label        text NOT NULL,
    icon         text,
    target_type  text NOT NULL,    -- view | url | dashboard
    target_id    uuid,
    "order"      integer NOT NULL DEFAULT 0,
    role_filter  text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Helpful indexes for generic list queries.
CREATE INDEX users_tenant_idx        ON users(tenant_id);
CREATE INDEX roles_tenant_idx        ON roles(tenant_id);
CREATE INDEX applications_tenant_idx ON applications(tenant_id);
CREATE INDEX modules_app_idx         ON modules(app_id);
CREATE INDEX entities_app_idx        ON entities(app_id);
CREATE INDEX fields_entity_idx       ON fields(entity_id);
CREATE INDEX relationships_from_idx  ON relationships(from_entity_id);
CREATE INDEX relationships_to_idx    ON relationships(to_entity_id);
CREATE INDEX indexes_entity_idx      ON indexes(entity_id);
CREATE INDEX views_entity_idx        ON views(entity_id);
CREATE INDEX menus_app_idx           ON menus(app_id);

COMMIT;
