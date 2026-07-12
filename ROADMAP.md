# Roadmap — Agentic SDLC Platform

Phased delivery. Each milestone is independently demoable and ends with a
working `docker compose up`. The early milestones prioritize the **model-driven
core** — because once that exists, the platform can start building itself.

---

## M0 — Bootstrap  ⬜  (~3 days)
Goal: runnable empty platform.
- [ ] Repo scaffold (monorepo layout, PLAN §8)
- [ ] `docker-compose.yml` infra (postgres×2, nats, redis, qdrant, vault,
      registry, otel stack, grafana)
- [ ] `shared/proto` event schemas + `shared/sdk-*` skeletons
- [ ] Base images (python/go/node) with OTel + sdk baked in
- [ ] `make up/down/logs/test`
**Demo:** `make up` → healthy Grafana + NATS; nothing else.

## M1 — Model-Driven Core  ⬜  (~1.5 weeks)  ⭐ foundation
Goal: define an app purely as metadata and see it work.
- [ ] `metadata-api`: typed CRUD over Layers 0–3 (tenants, users, apps, entities,
      fields, relationships, views, menus)
- [ ] `ddl-engine`: diff metadata → idempotent SQL migrations → apply to Tenant DB
- [ ] `data-api`: generic CRUD over any entity (validation, refs, filters, RLS stub)
- [ ] `permissions-engine`: role × entity × action (field/row-level later)
- [ ] `ui-registry` + `portal` generic renderer (list/form/detail/nav)
- [ ] Seed the **Users** app as pure metadata; manage platform users through it
**Demo:** Add an "Asset" entity via API → portal shows Asset list/form instantly.

## M2 — Identity + Requests + Workboard  ⬜  (~1 week)
Goal: stakeholders sign up, file requests, orchestrator routes them.
- [ ] `identity`: username/password signup/login (JWT), tenants, roles
- [ ] `gateway`: Traefik routing + JWT forwarding
- [ ] Seed **Support** + **PMO** apps (Request, Project, Task) as metadata
- [ ] Add `temporal` service to infra (persistence on Postgres); wire orchestrator
- [ ] `orchestrator`: Temporal skeleton; `request.created` → create project/tasks
- [ ] `notifications`: in-app notifications (portal bell)
- [ ] Workboard view in portal (Track A!)
**Demo:** Sign up → file "add timesheet app" → see routed project in workboard.

## M3 — First personas + self-build  ⬜  (~1.5 weeks)  ⭐ magic moment
Goal: AI delivers a new app end-to-end as metadata.
- [ ] `llm-gateway` (LiteLLM) on **DeepSeek V4 Flash**: structured outputs, retry-on-parse-fail, cost metering
- [ ] Weak-model safety: self-check step + independent **Reviewer** persona + reject→refine loop
- [ ] `workforce-api` + `agent-runtime` pool + persona injection
- [ ] Seed personas: L1 Support, Product Manager, Business Analyst,
      Solution Architect, Metadata Engineer, Test Engineer, Release Manager,
      Reviewer
- [ ] `knowledge` RAG seeded over the metadata dictionary
- [ ] Orchestrator wires intake → requirements → design → author metadata →
      validate → publish (with spec + publish approval gates)
**Demo:** File "we need an Asset Management app" → AI designs + builds it →
appears live in portal. No code written.

## M4 — Workflow / Rules / Permissions engines  ⬜  (~1 week)
Goal: full dynamic business logic.
- [ ] `workflow-engine` (states/transitions/actions/SLAs) interpreting metadata
- [ ] `rules-engine` (CEL/JSONLogic) on CRUD/event/cron triggers
- [ ] Field-level + row-level security in `permissions-engine`
- [ ] Kanban/calendar/dashboard views in renderer
**Demo:** Asset lifecycle workflow + "on employee exit → return assets" rule.

## M5 — Code track (Track B)  ⬜  (~2 weeks)
Goal: AI ships real services via sandbox + VCS.
- [ ] `vcs` (git repo mgmt, branches, PRs)
- [ ] `sandbox` ephemeral devboxes (Docker, allow-listed egress)
- [ ] Personas: Backend/Frontend/Platform Engineer, SDET, Security Analyst, SRE
- [ ] Custom-widget pipeline (`services/widgets/` → published to portal)
- [ ] Full SDLC: design → code → test → review → merge → deploy (gated)
**Demo:** Request a custom 3D model viewer widget → AI builds, tests, ships it.

## M6 — Support & continuous loop  ⬜  (~1 week)
Goal: incidents → auto-requests → improvement.
- [ ] Monitor (SRE) persona: scrape Prom/tail Loki, SLO breach → incident
- [ ] L2 Support persona: diagnose, reproduce, escalate, write knowledge articles
- [ ] Auto-file requests from incidents; link them to projects
- [ ] Persona evals: closed tasks → RAG + prompt regression suite
**Demo:** Kill a service → incident → auto-request → AI fixes → ships → resolved.

## M7 — Scale & governance  ⬜  (ongoing)
- [ ] True multi-tenant isolation + per-tenant cost budgets (`cost_ledger`)
- [ ] CTO persona: portfolio prioritization, workforce right-sizing, model mix
- [ ] Compliance/Audit persona: policy enforcement, data classification
- [ ] Blue/green + canary deploys; full rollback automation
- [ ] k8s/Helm for prod runtime (optional)

---

## Effort estimate
~8–10 weeks (1–2 engineers) to **M5** = a platform that builds both metadata
and code apps autonomously. M6–M7 turn it into a production IT department.
