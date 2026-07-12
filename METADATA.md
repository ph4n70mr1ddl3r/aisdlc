# Metadata Model — the heart of the platform

> Everything is metadata. The runtime reads these tables and *becomes* the
> application. Agents author rows here to deliver new capabilities (Track A).
> This file is the canonical data dictionary.

Principles:
- **Single source of truth**: every screen, field, rule, role, workflow, and
  AI persona is a row in these tables.
- **Versioned & publishable**: drafts → publish → migration → live. Rollback =
  revert to a prior published version.
- **Tenant-scoped**: most tables carry `tenant_id`; system metadata is global.
- **Two DBs**: a **metadata DB** (this dictionary) and a **tenant data DB**
  (real tables created on demand by `ddl-engine`).

---

## Layer 0 — Tenancy & Identity

| Table | Key columns | Notes |
|---|---|---|
| `tenants` | id, name, slug, plan | Orgs |
| `users` | id, tenant_id, username (unique), password_hash, name, status | Platform users (stakeholders); username/password login |
| `sessions` | id, user_id, token, expires | |
| `roles` | id, tenant_id, name, is_system | e.g. Admin, Stakeholder, Maintainer |
| `role_assignments` | user_id, role_id, scope | scope = app/entity/record filter |

---

## Layer 1 — Application Model

| Table | Key columns | Notes |
|---|---|---|
| `applications` | id, tenant_id, name, slug, icon, version, status(draft/published), description | A bundled set of entities/UIs/workflows. System apps seeded: `admin`, `users`, `support`, `pmo`, `devconsole` |
| `modules` | id, app_id, name, order | Grouping within an app |
| `app_dependencies` | app_id, depends_on_app_id | Apps can reuse entities from others |

---

## Layer 2 — Data Model (dynamic schema)

| Table | Key columns | Notes |
|---|---|---|
| `entities` | id, app_id, name, table_name, label_singular, label_plural, icon, is_audit, is_system | Maps to a real table in tenant DB |
| `fields` | id, entity_id, name, label, **type**, config(json), order, is_system | `type` see below |
| `relationships` | id, name, from_entity_id, to_entity_id, kind(oneToMany/manyToOne/manyToMany), inverse_name | Foreign keys / join tables |
| `indexes` | id, entity_id, fields[], unique | |
| `validations` | id, entity_id OR field_id, expr, message | Record/field-level rules |

**Field types:** `text, longtext, number, decimal, bool, date, datetime, time,
enum, multiselect, ref, multiref, money, email, url, phone, json, file, image,
computed, formula, agentref, userref, richdoc`.

**Computed vs formula:** `formula` is evaluated by `data-api` on read from the
declared expression; `computed` is re-derived on write. Both run in the
application layer (never the DB) so they stay tenant-aware and permission-checked.

**Field config** (JSON) holds: `required, unique, default, validation_expr,
ui_widget, ui_options, placeholder, pii, encrypted, indexed, ref_entity,
ref_display_field, enum_options, formula_lang`.

**Physical storage:** `ddl-engine` creates one real table per entity (typed
columns) + a `meta jsonb` overflow column for sparse/extra fields. Refs become
real FKs (within the tenant DB). This gives speed + query power while staying
flexible.

**Cross-database references:** `userref`/`agentref` point into the **metadata
DB**, which is a separate logical database from the tenant data DB, so they
cannot be physical FKs. They are **validated soft references** — resolved and
authorization-checked by `data-api` + `permissions-engine` at read/write time.

---

## Layer 3 — UI Model (rendered by the portal)

| Table | Key columns | Notes |
|---|---|---|
| `views` | id, entity_id, **type**, name, config(json), is_default | type ∈ `list, form, detail, kanban, calendar, gallery, dashboard` |
| `list_configs` (inside views) | columns[], default_filters, default_sort, search_fields, page_size, bulk_actions | |
| `form_configs` | sections[], field_order, conditional_visibility, layout | |
| `detail_configs` | tabs[], related_lists[] | |
| `menus` | id, app_id, parent_id, label, icon, target_type, target_id, order, role_filter | Tree → navigation |
| `dashboards` | id, app_id, name, widgets[] | widgets ref views/metrics |
| `actions` | id, entity_id, name, scope(record/bulk/global), kind(workflow/api/script/ui), target_id, confirm | Buttons on views |
| `reports` | id, entity_id, name, query, chart_type, schedule | |

**Action kinds:** `workflow` fires a transition; `api` calls an HTTP endpoint
recorded in metadata; `ui` is client-side navigation/modal. `script` is **not**
arbitrary runtime code — it names a vetted, allow-listed routine registered via
Track B and runs sandboxed; it cannot be created by a Track-A metadata edit
alone. Anything beyond the safe DSL (Layer 5) is Track B.

The portal is a **schema-driven renderer**: give it a `view` row, it renders a
screen. Zero hardcoded business screens. Custom visuals beyond the renderer's
vocabulary fall to Track B (`widgets/`).

---

## Layer 4 — Workflow / Process Model

| Table | Key columns | Notes |
|---|---|---|
| `workflows` | id, name, **target_kind**(entity/request_type/incident), target_id, version | State machine definition |
| `states` | id, workflow_id, name, is_initial, is_terminal, on_enter(json), on_exit(json), sla_minutes | |
| `transitions` | id, workflow_id, from_state_id, to_state_id, guard_expr, actions(json), approver_role_id | |
| `slas` | id, state_id, warn_minutes, breach_minutes, escalate_to | |
| `timers` | id, workflow_id, cron_or_delay, action | |

**Timers** (`cron_or_delay`) are owned and fired by `workflow-engine`, which
runs the scheduler for every published workflow.

`actions` are declarative: `{type: call_persona|set_field|send_notif|run_rule|create_record|call_api, ...}`.
The `workflow-engine` interprets these; no code per workflow.

---

## Layer 5 — Rules Engine

| Table | Key columns | Notes |
|---|---|---|
| `rules` | id, name, trigger, condition_expr, action(json), priority, enabled | trigger ∈ `beforeInsert, afterInsert, beforeUpdate, afterUpdate, onDelete, onEvent, cron` |
| `rule_logs` | id, rule_id, record_ref, result, ts | For debugging |

Expressions use a safe DSL (e.g. CEL or JSONLogic) — never raw code.
**Cron/event triggers** are dispatched by `rules-engine` (cron scheduler) and
the NATS bus (event triggers); there is no per-rule daemon.

---

## Layer 6 — Security & Permissions

| Table | Key columns | Notes |
|---|---|---|
| `permissions` | role_id, entity_id, actions(CRUD+custom) | e.g. role "Agent" can't read "Salary" |
| `field_permissions` | role_id, field_id, read, write | Field-level masking |
| `row_level_security` | id, role_id, entity_id, scope_expr | "only see records where owner = me" |
| `api_keys` | id, tenant_id, scopes | For integrations |

`permissions-engine` enforces all three on every `data-api` call and the portal
hides UI accordingly.

---

## Layer 7 — AI Workforce (also metadata!)

The AI org is itself metadata — so it can be reorganized, grown, and tuned
without code. See PERSONAS.md for the full persona library.

| Table | Key columns | Notes |
|---|---|---|
| `departments` | id, parent_id, name, mission | Org tree: Product, Engineering, QA, DevOps, Security, Support, Data, PMO, Governance |
| `personas` | id, name, department_id, system_prompt, toolset[], model_pref(default deepseek-v4-flash), kpis(json), cost_budget, can_wear | The "hats" (PM, Architect, BackendDev…). All default to DeepSeek V4 Flash |
| `agents` | id, name, status(idle/busy), current_persona_id, capacity, tenant_id | Generic workers in the pool |
| `assignments` | agent_id, persona_id, task_id, started_at | Who's wearing what, when |
| `raci_templates` | request_type, role/persona → R/A/C/I | Auto-staffing of projects |

---

## Layer 8 — Work Tracking (the "Jira/ServiceNow" layer)

These are themselves metadata-defined entities, seeded as system apps:

| Table | Key columns | Notes |
|---|---|---|
| `request_types` | id, name (feature/bug/support/infra/data/change/access), default_workflow_id, sla | |
| `requests` | id, type_id, title, description, requester_id, status, priority, project_id?, tenant_id | Anything anyone asks for |
| `projects` | id, name, goal, sponsor_id, status, method(scrum/kanban), roadmap | A delivery vehicle for a request |
| `epics` | id, project_id, title, order | |
| `stories` | id, epic_id, title, acceptance_criteria, persona_owner, points | |
| `tasks` | id, story_id OR request_id, persona_id, agent_id, status, estimate, trace_id | Unit of agent work |
| `sprints` | id, project_id, goal, start, end, stories[] | If scrum |
| `incidents` | id, service, severity, status, root_cause, linked_request_id | ITIL |
| `problems` | id, linked_incidents[], status | |
| `changes` | id, type(standard/normal/emergency), risk, approval, deployment_id | ITIL change mgmt |
| `knowledge_articles` | id, title, body, tags | Support self-service |

---

## Layer 9 — Delivery & Build

| Table | Key columns | Notes |
|---|---|---|
| `metadata_bundles` | id, app_id, version, entities[], fields[], views[], workflows[], rules[] | A snapshot of a deliverable |
| `builds` | id, bundle_id OR repo_ref, type(metadata|code), status, artifacts | |
| `deployments` | id, build_id, env(dev/staging/prod), status, approver_id, rollback_to | |
| `migrations` | id, deployment_id, up_sql, down_sql, status | Generated by ddl-engine from metadata diff |

---

## Layer 10 — Audit & Cost

| Table | Key columns | Notes |
|---|---|---|
| `audit_log` | id, actor(user/agent/persona), action, target_ref, before, after, ts, trace_id | Append-only, hashed |
| `agent_runs` | id, task_id, agent_id, persona_id, status, tokens_in/out, cost_usd, duration, trace_id, verify_status, retries | verify_status = self_check / reviewed |
| `reviews` | id, run_id, reviewer_persona_id, verdict(approve/reject), notes, ts | Independent reviewer pass (weak-model safety) |
| `cost_ledger` | id, tenant_id, project_id, persona_id, tokens, usd, ts | Per-tenant budgeting |
| `metrics` | id, ref, name, value, ts | Usage telemetry |

---

## Lifecycle of a metadata change

```
draft (in metadata-api) ─► validate (schema lint + perms check)
                          ─► review (Architect persona / human gate)
                          ─► BUILD: snapshot to metadata_bundles
                          ─► ddl-engine computes diff vs current → migration
                          ─► DEPLOY to dev ─► test ─► staging ─► prod (approval)
                          ─► PUBLISH: portal picks up new views/menus instantly
                          ─► audit_log + (rollback = redeploy prior bundle)
```

Because publishing is metadata-only, **Track A deploys are seconds, not minutes.**
