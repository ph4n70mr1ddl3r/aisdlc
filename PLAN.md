# Agentic SDLC Platform — Master Plan

> **Vision:** An AI-run IT department delivered as a service. Stakeholders log in,
> request anything (a feature, a bug fix, support, a whole new application), and
> an **autonomous AI workforce** wearing many hats runs the entire lifecycle —
> intake, design, build, ship, support, monitor. The platform is **model-driven**:
> users, workflows, UIs, rules, roles — even the AI org chart — live as metadata
> in the database. Because new capabilities are *just metadata*, **the platform
> builds and extends itself.**

---

## 1. What this is (the one-paragraph pitch)

It's **Salesforce + Jira + ServiceNow + an SDLC pipeline**, except every seat in
the "company" — CTO, PM, BA, Architect, Developer, QA, DevOps, Security,
Support, Data — is staffed by an AI agent that can wear any hat. A request from
a user becomes a tracked project; the right AI personas pick it up; the result
is delivered as **metadata** (so it's live instantly) or, when needed, as real
code shipped through a full agent-driven SDLC. Then Support agents monitor it
and file the next round of requests. **It's one whole IT department, run by AI,
that grows itself.**

---

## 2. The four pillars

| Pillar | What it means | Where it lives |
|---|---|---|
| **P1 Model-Driven Core** | Everything is metadata: data models, UIs, workflows, rules, roles, menus. Runtime interprets it. No hardcoded business screens. | `METADATA.md` |
| **P2 Self-Building Platform** | New capability = new metadata. Agents author metadata → publish → it's live. Platform extends itself without redeploys. | `LIFECYCLE.md` §4 |
| **P3 AI Workforce (multi-hat)** | A pool of generic agent workers; each task assigns a *persona* (hat): prompt + tools + KPIs. Org chart is metadata. | `PERSONAS.md` |
| **P4 Full IT Department** | Not just dev. Covers PMO, Engineering, QA, DevOps, Security, Support, Data, Governance — every function tracked & served. | `LIFECYCLE.md`, `PERSONAS.md` |

---

## 3. Architecture overview

```
                         ┌───────────────────────────────────┐
                         │       Stakeholder Portal           │  generic React
                         │  (RENDERS UI FROM METADATA)        │   renderer
                         └─────────────────┬─────────────────┘
                           ┌───────────────┴───────────────┐
                 ┌─────────▼─────────┐             ┌────────▼────────┐
                 │   Metadata API    │             │   Data API      │
                 │  (dictionary CRUD)│             │ (generic CRUD   │
                 └─────────┬─────────┘             │  over any entity)│
                           │                       └────────┬────────┘
   ┌───────────────────────┼─────────────┐                   │
   ▼            ▼          ▼             ▼                   │
[Workflow   [Rules     [Permissions   [UI/Menu             [DDL/Migration
 Engine]    Engine]     Engine]        Registry]            Engine]
   │                                          │
   └──────────► actions dispatch ◄────────────┘
                         │
        ┌────────────────┴─────────────────┐
        ▼                                  ▼
 ┌──────────────┐                   ┌──────────────┐
 │ Orchestrator │ ── assign ───────► │  Agent Pool  │ (workers)
 │ (workboard)  │ ◄── results ────── │  + PERSONAS  │ (hats)
 └──────┬───────┘                   └──────┬───────┘
        │ dispatch tasks / personas         │ call
        ▼                                    ▼
 ┌──────────────────────────┐         ┌──────────────┐
 │ LLM Gateway │ RAG │ VCS  │         │  Sandbox     │
 │ Vault │ Sandbox │ Tools │         │ (code track) │
 └──────────────────────────┘         └──────────────┘

Event bus (NATS) + OTel everywhere + Postgres (metadata) + Postgres (tenant data)
+ Qdrant (RAG) + Vault (secrets) + Docker (everything)
```

Two **delivery tracks** flow out of the workforce:
- **Track A — Metadata delivery** (no-code, instant): ~80% of requests → agents author entities/fields/views/workflows/rules → publish → live in the portal.
- **Track B — Code delivery** (full SDLC): new platform services, custom widgets, integrations → real repos, PRs, CI/CD (the original agentic SDLC).

---

## 4. Microservices catalog

### 4.1 Model-Driven Core (the heart)
| Service | Role |
|---|---|
| `metadata-api` | CRUD over the entire dictionary (entities, fields, views, workflows, rules, roles, personas…) |
| `data-api` | Generic CRUD over *any* entity by reading its field metadata; validation, refs, row-level security |
| `ddl-engine` | Generates + applies SQL migrations from metadata diffs (add entity → CREATE TABLE, etc.) |
| `workflow-engine` | Interprets workflow metadata; state machines per record/request; fires actions |
| `rules-engine` | Evaluates business rules on triggers (before/after insert/update, events, cron) |
| `permissions-engine` | Role × entity × action + field-level + row-level security; gates every call |
| `ui-registry` | Serves view/form/menu/dashboard definitions to the renderer |

### 4.2 Platform services
| Service | Role |
|---|---|
| `gateway` | Edge routing, auth, rate limit |
| `identity` | Users, tenants, roles, sessions, OAuth |
| `portal` | **Generic UI renderer** (reads metadata → React screens). Pre-seeded system apps: Admin, Users, Support, PMO, Developer Console |
| `notifications` | Email/in-app/Slack/webhooks |
| `orchestrator` | Workboard; routes requests → projects → tasks → personas (Temporal-backed from M2; NATS/DB-driven bootstrap) |
| `llm-gateway` | Provider abstraction (LiteLLM), quotas, cost metering |
| `knowledge` | RAG over metadata + code + docs (Qdrant) |
| `vcs` | Git repo, branch, PR mgmt (Track B) |
| `sandbox` | Ephemeral Docker dev envs (Track B) |
| `audit` | Append-only audit log of every privileged action |

> ℹ️ **Secrets** (HashiCorp Vault) and **observability** (OTel Collector +
> Grafana/Loki/Tempo/Prometheus) are **infrastructure**, not app services —
> see section 1 of `docker-compose.yml`. That makes platform = **10** app
> services, total = **19** (7 core + 10 platform + 2 workforce).

### 4.3 AI workforce
| Service | Role |
|---|---|
| `agent-runtime` | The ReAct worker; runs tasks wearing an assigned persona (this is a *pool*, scaled horizontally) |
| `workforce-api` | Manages `agents` (workers), `personas` (hats), `departments`, capacity, cost ledger |

> ⚠️ Old plan had one service per fixed phase-agent (triage, dev, qa…). Now there
> is **one runtime service** plus a **persona library** in the DB. See PERSONAS.md.

---

## 5. The self-building loop (the differentiator)

```
  User: "We need an Asset Management app"
        │
        ▼
  ┌─ Request logged (type=feature) ──────────────────────────┐
  │  PM persona → scope       Architect persona → design      │
  │  (entities Asset/Location/Assignment, fields, relations,  │
  │   list+form+detail views, Asset-lifecycle workflow,       │
  │   rules, roles)                                           │
  └───────────────────────────────────────────────────────────┘
        ▼
  ┌─ Developer personas AUTHOR METADATA (JSON) ──────────────┐
  │  QA persona validates via generic renderer (no code)      │
  │  Migration persona → ddl-engine creates tables            │
  └───────────────────────────────────────────────────────────┘
        ▼
  PUBLISH ──► Asset Management app appears in the user's portal
              instantly: full CRUD, workflows, dashboards, roles.
        ▼
  Support persona onboards user, monitors usage, files improvement
  requests ──► (loop)
```

Most requests never touch a compiler. Code (Track B) is reserved for new
platform services, custom UI widgets the renderer can't express, heavy
integrations, and performance work.

---

## 6. Stakeholder experience

1. **Sign in** → portal shows apps they have access to (system + custom-built).
2. **Ask for anything** via:
   - a guided **Request** form (feature/bug/support/infra/data/access/change), or
   - natural-language **chat** → Support persona converts it to a request.
3. **Track it** like a real IT ticket: status, assignee persona, phase, logs,
   estimated delivery, cost.
4. **Approve at gates**: spec sign-off, UAT, prod deploy (configurable per
   request type / risk).
5. **Use the result**: new app/module is live in their portal; or a fix shipped.
6. **Get support**: incidents auto-detected by Monitor persona, or filed manually;
   routed through Support → Engineering as needed.

Every interaction is also just records in metadata-defined entities — Support,
Requests, Projects, Incidents are themselves apps the platform ships to itself.

---

## 7. Tech stack

| Layer | Choice |
|---|---|
| Portal (renderer) | Next.js + React + Tailwind; a **schema-driven** form/list/detail engine |
| Core/platform services | Go (gin) |
| Workflow/rules/agent/LLM services | Python (FastAPI) |
| Orchestration | Temporal.io (from M2; NATS JetStream bootstrap) |
| Event bus | NATS JetStream |
| Metadata DB | PostgreSQL (the dictionary) |
| Tenant data DB | PostgreSQL (dynamically-created tables per entity, + JSONB overflow) |
| RAG | Qdrant + LangChain |
| LLM abstraction | LiteLLM |
| Cache/queue | Redis |
| Sandbox | Docker-in-Docker (Track B) |
| Secrets | Vault |
| Observability | OpenTelemetry + Grafana stack |
| Containers | Docker Compose (dev), Swarm/k8s (prod) |

---

## 8. Repository layout

```
aisdlc/
├── docker-compose.yml
├── README.md  PLAN.md  ARCHITECTURE.md  METADATA.md  PERSONAS.md
├── LIFECYCLE.md  ROADMAP.md  SETUP.md
├── shared/          # proto (event schemas), sdk-py/go/ts, tool registry
├── services/
│   ├── core/        # metadata-api, data-api, ddl-engine, workflow-engine,
│   │                # rules-engine, permissions-engine, ui-registry
│   ├── platform/    # gateway, identity, portal, notifications, orchestrator,
│   │                # llm-gateway, knowledge, vcs, sandbox, secrets, audit
│   ├── workforce/   # agent-runtime, workforce-api
│   └── widgets/     # custom UI widgets beyond the generic renderer (Track B)
├── personas/        # versioned persona prompts + tool sets + KPIs (seed data)
├── seed/            # bootstrap metadata: system apps (Users, Admin, Support, PMO)
└── infra/           # postgres, nats, qdrant, vault, otel, ...
```

---

## 9. Roadmap (summary — see ROADMAP.md)

- **M0 Bootstrap** — repo, compose, infra, shared SDKs
- **M1 Model-Driven Core** — metadata-api, data-api, ddl-engine, generic portal renderer; ship the **Users app** defined purely as metadata
- **M2 Identity + Requests + Workboard** — stakeholders can sign up, file requests, orchestrator routes them
- **M3 First personas** — PM + Architect + Developer personas author metadata end-to-end (self-build demo)
- **M4 Workflow/Rules/Permissions engines** — full dynamic business logic
- **M5 Code track (Track B)** — developer/QA/SRE personas ship real services via sandbox + VCS
- **M6 Support loop** — Monitor + Support personas, incidents, continuous improvement
- **M7 Scale** — multi-tenant, cost governance, self-reorg of AI workforce

---

## 10. Open decisions

- [ ] Data storage: real-tables-per-entity (fast, typed) vs JSONB-first (flexible) vs hybrid → **lean hybrid**.
- [ ] UI renderer: build bespoke vs adopt a low-code lib (Form.io, React-Admin schema) → build thin, own the metadata shape.
- [ ] Workflow engine: Temporal (code) vs fully metadata-driven BPMN (Camunda) → Temporal + a metadata DSL it interprets.
- [ ] Persona licensing/cost: per-tenant model budgets and quotas.
- [ ] Governance: who can publish metadata to prod? (approval gates, sandbox tenants)
