# Architecture — Agentic SDLC Platform

Companion to [PLAN.md](./PLAN.md). Deep dive on the model-driven core, the
runtime engines, the agent runtime, and control flow.

---

## 1. The two databases

- **Metadata DB** (Postgres) — the dictionary (see METADATA.md). Source of truth
  for *what the system is*. Versioned, publishable, tenant-scoped.
- **Tenant Data DB** (Postgres) — *business data*. Tables are created on demand
  by `ddl-engine` (one real table per entity + JSONB overflow). Typed columns =
  fast, queryable; overflow = flexible.

Both managed by the same Postgres cluster, separate logical DBs, separate
connection pools, separate backup schedules.

---

## 2. The model-driven core (runtime engines)

```
                    ┌──────────────────────────────────────────┐
                    │              METADATA DB                  │
                    └─────────────────────┬────────────────────┘
                                          │ read
   ┌──────────────┬──────────────┬────────┴─────────┬──────────────┬──────────────┐
   ▼              ▼              ▼                  ▼              ▼              ▼
metadata-api   data-api     ddl-engine      workflow-engine   rules-engine   permissions
 (CRUD dict)   (generic     (DDL from       (state machines   (triggered     -engine
                CRUD over    metadata diff)  from workflows)   expressions)   (role/field/
                any entity)                                                   row security)
   │              │                                              │              │
   └──────────────┴──────────────┬───────────────────────────────┴──────────────┘
                                  ▼
                          ui-registry ──► Portal (renderer)
```

- **metadata-api**: typed CRUD + validation + versioning over every dictionary
  table. Drafts → publish.
- **data-api**: given an entity name, reads its `fields` metadata and exposes
  CRUD with validation, refs, filters, sorting, pagination, RLS. *The same
  endpoint serves every entity.*
- **ddl-engine**: on publish, diffs the new metadata vs current, emits idempotent
  `up/down` SQL migrations, applies them to the Tenant Data DB, records a
  `migrations` row.
- **workflow-engine**: interprets `workflows`/`states`/`transitions`; moves
  records through states; runs `actions` (call persona, set field, notify, run
  rule, create record, call API); enforces SLAs.
- **rules-engine**: evaluates `rules` on triggers (before/after CRUD, events,
  cron) using a safe DSL (CEL/JSONLogic).
- **permissions-engine**: gates every data-api + UI element from
  `permissions`/`field_permissions`/`row_level_security`.
- **ui-registry**: serves `views`/`menus`/`dashboards`/`actions` to the portal.

These engines are **generic** — they have no knowledge of specific apps. All
app behavior emerges from metadata. This is what makes the platform
self-extending.

---

## 3. The Portal is a renderer, not a UI

`portal` (Next.js) ships a fixed set of generic components:
`ListView, FormView, DetailView, KanbanView, CalendarView, Dashboard,
RecordRef, FieldEditor (per type), ActionMenu, NavTree`.

At runtime it asks `ui-registry` for the view/menu and renders. Adding a new
screen = adding metadata. (System apps — Admin, Users, Support, PMO,
DevConsole — are seeded metadata, not bespoke code.) Anything the renderer
can't express is a custom widget in `services/widgets/` via Track B.

---

## 4. AI workforce runtime

One service, `agent-runtime`, runs a pool of workers. The orchestrator hands a
task a **persona** (prompt + tools + budget + KPIs) — see PERSONAS.md.

ReAct loop per task (all on **DeepSeek V4 Flash**):
```
inject persona + context(task, rag_hits)
loop until done | KPIs met | budget exhausted:
    LLM(thought) ─► action(tool, args)   # only allow-listed tools
    execute tool (sandboxed, traced)     ─► observation
    append to trace
self-check (schema + KPIs + lint) ─► retry on failure (max N), else escalate
emit task.finished { artifacts, tokens, cost, kpi_results }
        │
        ▼
Reviewer persona (separate task) ─► APPROVE | REJECT-with-notes ─► loop back
```
- Tools are capability-scoped per persona and authorized by a signed capability
  token minted by the orchestrator per task.
- Track-A tools write metadata (via metadata-api). Track-B tools use the
  sandbox (git/shell/build).
- Every LLM call requests a JSON schema; the gateway rejects + retries on parse
  failure. No model output is published/merged without self-check **and** an
  independent Reviewer (see §11).
- Every run → `agent_runs` (trace_id, tokens, cost, verify_status) + OTel spans.

---

## 5. Orchestrator / workboard

Built on **Temporal** for durable workflows (Temporal lands in **M2**; the
M0–M1 bootstrap drives orchestration via NATS JetStream + the metadata DB):
- Subscribes to `request.created`; loads the `raci_template` for its type.
- Creates `project`/`stories`/`tasks`; assigns personas to free agents.
- Drives each task through its workflow states; pauses on human gates
  (Temporal Signals); enforces SLAs + escalations.
- Emits events on the bus: `task.assigned`, `task.finished`, `gate.waiting`,
  `deployment.done`, `incident.raised`.

State lives in the metadata DB (`requests`, `tasks`, `deployments`, …); Temporal
holds only the workflow orchestration state.

---

## 6. Event bus & contracts

NATS JetStream. Streams: `requests`, `projects`, `tasks`, `metadata.published`,
`workflows.transitioned`, `deployments`, `incidents`, `agent.runs`,
`approvals`. Canonical event envelope + JSON Schema in `shared/proto`:
```json
{ "id":"uuid","stream":"tasks","type":"task.finished",
  "ts":"…","trace_id":"…","subject":"task:1234","payload":{…},"version":1 }
```
Consumers are idempotent (dedupe on `id`, order on `subject`).

---

## 7. Sandbox (Track B)

`sandbox` spins ephemeral Docker devboxes for code-track personas:
```
agent ──► sandbox.create({repo, branch}) ──► ws_url
       ──► sandbox.exec(ws_url, ["pytest","build",...])
       ──► sandbox.destroy(ws_url)
```
No egress except an allow-list proxy (registry, git, llm-gateway). CPU/mem/time
quotas. Source of truth stays in git.

---

## 8. LLM gateway, RAG, secrets

- **llm-gateway** (LiteLLM): powers the whole workforce on **one** model —
  **DeepSeek V4 Flash**. Enforces structured/schema-validated outputs, retries
  on parse failure, per-tenant keys/quotas, cost metering, streaming +
  tool-calling, versioned prompts. Quality comes from *process* (§11), not model
  size. A stronger reviewer model can be wired for the review pass only (off by
  default).
- **knowledge** (Qdrant + LangChain): embeds metadata + code + docs; on
  `pr.merged`/`metadata.published` re-indexes changed chunks. Personas call it
  to ground decisions.
- **secrets** (Vault): per-project/per-env credentials; never in agent env.

---

## 9. Observability & audit

- **OpenTelemetry** end-to-end (Collector → Tempo/Loki/Prom/Grafana).
- Per-request "flight recorder": timeline of personas, tokens, cost, diffs,
  test results, approvals.
- **audit** service: append-only, hashed log of every privileged action
  (metadata publish, prod deploy, secret access) for compliance.

---

## 10. Failure modes & mitigations

| Risk | Mitigation |
|---|---|
| Bad metadata bricks an app | Draft→validate→review→publish; instant rollback to prior bundle |
| Agent hallucinated API/data | Permissions + validation gate every data-api call; tests + reviewer |
| Runaway cost | Per-task budget (tokens/$/time) hard cap + tenant quotas in cost_ledger |
| Bad prod change | ITIL change record + approval gate; one-click rollback; blue/green |
| Secret leak | Vault, no agent env secrets, sandbox egress allow-list |
| Weak model hallucinates / low-quality output | Self-check + schema validation + retries; independent Reviewer must APPROVE before publish/merge; tests + human gate; reject loops back with notes |
| LLM outage | llm-gateway retry/backoff; optional stronger reviewer model for hard cases |
| Workflow stuck on approval | SLA timers + escalation to maintainer |
| Self-build loops forever | Max iterations per project; CTO persona re-prioritizes; human escalation |

## 11. Weak-model safety: verify → review → gate

The entire workforce runs on **one** model — **DeepSeek V4 Flash** — chosen for
cost and speed, not capability. A weaker model is safe **only** because no
single model output is ever trusted. Every artifact runs a quality gauntlet:

1. **Structured outputs only.** Every LLM call requests a JSON schema (entities,
   migrations, code diffs, tool args). The gateway rejects + retries (max N) on
   schema/parse failure; persistent failure escalates to a human.
2. **Self-check.** Before `task.finished`, the authoring persona re-evaluates its
   own KPIs (criteria coverage, tests pass, lint, no-placeholder). A failing
   self-check loops the task back with the failure as context.
3. **Independent reviewer.** A *different* persona (Reviewer/Architect on Track A;
   SDET + Security on Track B) inspects the artifact blind to the author's
   reasoning and returns `APPROVE | REJECT-with-notes`. Reject loops back.
4. **Objective gates.** Tests run in the sandbox; metadata is validated by
   ddl-engine and exercised by the portal renderer; permissions gate every
   data-api call. These are deterministic and don't depend on the model.
5. **Mandatory human gates.** Spec sign-off, design review, change approval,
   UAT, prod deploy, and anything touching secrets/prod-data/external systems
   always needs a human (PERSONAS.md §7).
6. **Circuit breaker.** A task that rejects/refines past its budget escalates
   instead of looping forever.

Net effect: the cheap model produces a draft; the *system* guarantees quality.
Each pass is recorded in `reviews` and `agent_runs.verify_status` for audit.
