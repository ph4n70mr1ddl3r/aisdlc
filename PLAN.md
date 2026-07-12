# Agentic SDLC Platform — Master Plan

> **Vision:** A platform where stakeholders (product, business, QA, ops) log in,
> file tickets, and an autonomous multi-agent system drives the **entire software
> development lifecycle** — requirements → design → code → review → test →
> release → monitor — looping back into the backlog. Humans approve at gates;
> AI does the rest.

---

## 1. Goals & Principles

| # | Goal | How |
|---|------|-----|
| G1 | Whole cycle owned by AI | One agent role per SDLC phase, orchestrated by a central planner |
| G2 | Stakeholders only file tickets + approve | Self-service portal, email/in-app approvals |
| G3 | Microservices | Each agent + each platform concern = its own service |
| G4 | Docker-first | Everything containerized; `docker compose up` runs the dev platform |
| G5 | Safe & observable | Sandboxed agent workspaces, full trace of every agent action |
| G6 | Human-in-the-loop at risk gates | Spec sign-off, PR merge, prod deploy require approval |

**Principles:** event-driven backbone, idempotent agents, retry with backoff,
explicit state machine per ticket, single source of truth = ticket state.

---

## 2. High-Level Architecture

```
                         ┌─────────────────────────────┐
                         │     Stakeholder Web Portal   │  (Next.js)
                         │  login · file ticket · approve│
                         └───────────────┬─────────────┘
                                         │ HTTPS
                         ┌───────────────▼─────────────┐
                         │       API Gateway            │  (Kong/Traefik)
                         │   auth · routing · rate-limit│
                         └───────────────┬─────────────┘
                                         │
        ┌────────────────────────────────┼────────────────────────────────┐
        │            Event Bus  (NATS / Kafka)                              │
        └────────────────────────────────┬────────────────────────────────┘
                                         │ events: ticket.created, spec.ready, ...
   ┌───────────┬───────────┬──────────────┼───────────────┬───────────┬───────────┐
   ▼           ▼           ▼              ▼               ▼           ▼           ▼
[Triage]  [Require-  [Architect]     [Developer]      [Reviewer]  [QA/Test]  [Deploy]
 Agent]    ments      Agent           Agent (+sandbox) Agent       Agent      Agent
   │       Agent                                                         │
   │                                                                      │
   └──────────────► Orchestrator (state machine) ◄───────────────────────┘
                          │
   ┌──────────────────────┼──────────────────────┐
   ▼                      ▼                      ▼
[LLM Gateway]      [Knowledge/RAG]         [VCS/Git Service]
(provider-agnostic)  (vector + codebase)   (branches, PRs, merges)

Cross-cutting: Identity · Notifications · Observability · Secrets · Audit
```

---

## 3. Microservices Catalog

Each row = one Docker service in `docker-compose.yml`. All expose REST + emit
events. Services own their own DB schema (database-per-service pattern).

### 3.1 Platform services
| Service | Responsibility | Tech | Store |
|---|---|---|---|
| `gateway` | Edge routing, auth checks, rate limit | Traefik/Kong | — |
| `identity` | Accounts, roles, OAuth, JWT, approvals | Go | Postgres `identity` |
| `portal` | Web UI + BFF for stakeholders | Next.js/TS | — |
| `backlog` | Tickets, backlog, priorities, linking | Go | Postgres `backlog` |
| `orchestrator` | Ticket state machine, dispatches agents | Python (Temporal) | Postgres `wf` |
| `vcs` | Git repo mgmt, branches, PRs, merges | Go + git/libgit2 | Postgres `vcs` |
| `llm-gateway` | Provider abstraction, cost/quota, routing | Python (LiteLLM) | Redis |
| `knowledge` | RAG over codebase + docs, embeddings | Python | Qdrant + S3 |
| `sandbox` | Ephemeral Docker dev envs for agents | Python + Docker socket | — |
| `notifications` | Email/in-app/Slack/webhooks | Node | Postgres `notif` |
| `secrets` | Vault for API keys, deploy creds | HashiCorp Vault | — |
| `observability` | Logs, traces, metrics of agents | OTel Collector + Grafana | Loki/Prom/Tempo |

### 3.2 Agent services (one per SDLC phase)
| Service | Role | Key actions |
|---|---|---|
| `agent-triage` | Product Owner | Classify ticket, dedupe, prioritize, route |
| `agent-requirements` | BA | Draft spec, ask clarifying Qs, propose acceptance criteria |
| `agent-architect` | Solution Architect | Tech design, file/plan, API contracts, risk notes |
| `agent-developer` | Engineer | Implement in sandbox, commit, open PR |
| `agent-reviewer` | Code Reviewer + Sec | Review PR, SAST, dependency check, request changes |
| `agent-qa` | QA / SDET | Generate + run tests, report coverage |
| `agent-deploy` | DevOps | Build image, push, deploy to staging→prod with approvals |
| `agent-monitor` | SRE | Watch metrics/logs, auto-file incident tickets, rollback |

Agents are thin services: consume a task event → call `llm-gateway` + tools
(git, shell, sandbox) → emit result event. The *intelligence* lives in
per-agent system prompts + tool sets + the orchestrator's plan.

---

## 4. The Agentic Cycle (Ticket State Machine)

```
 FILED ─► TRIAGED ─► SPEC_DRAFTING ─► SPEC_REVIEW ─► SPEC_APPROVED
   │          │            │              │(human gate)
   │          └────────────┴──────────────┘
   ▼
 ARCHITECTING ─► ARCH_REVIEW ─► DEVELOPING ─► REVIEW ─► TESTING ─► STAGING
                                    │           │          │
                                    └─reject──►─┴──reject─► DEVELOPING
   ▼
 UAT(human) ─► PROD_DEPLOY(human) ─► MONITORING ─► (loop) CLOSED
                                                    │
                                              incident? ─► FILED
```

**Human-in-the-loop gates (default ON, configurable per project):**
1. `SPEC_REVIEW` — stakeholders approve requirements
2. `ARCH_REVIEW` — tech lead (or auto) approves design
3. `PR_MERGE` — reviewer approval (or auto if agent confidence + tests green)
4. `UAT` — stakeholder acceptance on staging
5. `PROD_DEPLOY` — release approval

Each transition is an event; the `orchestrator` persists state and drives the
next agent. Approvals come via the portal, email links, or Slack.

---

## 5. Stakeholder Experience

1. **Sign up** at portal → `identity` issues JWT.
2. **File a ticket**: free-text ("login page is slow on mobile") or template.
3. `agent-triage` enriches it (type, severity, suggested epic).
4. `agent-requirements` posts a draft spec → stakeholder gets an email + portal
   task to **Approve / Request changes**.
5. Stakeholder tracks progress on a live pipeline view (phase, logs, diffs).
6. At UAT they get a staging URL + acceptance checklist → approve.
7. They approve prod deploy → notified on completion.
8. Post-release, `agent-monitor` reports health; any regression auto-creates a
   new ticket (loop closed).

**Roles:** `stakeholder` (file/approve), `maintainer` (config, override),
`viewer` (read-only).

---

## 6. Data Model (core entities)

```
Account(id, email, role, created)
Ticket(id, title, description, type, status, severity, reporter_id, project_id)
Spec(id, ticket_id, body, version, status)
Design(id, ticket_id, plan, contracts, status)
ChangeRequest(id, ticket_id, branch, pr_url, diff_url)
Build(id, change_request_id, image, status, env)
Approval(id, ref_type, ref_id, gate, decision, by, ts)
AgentRun(id, ticket_id, agent, status, trace_id, tokens, cost, ts)
Event(id, stream, type, payload, ts)   -- append-only event log
```

---

## 7. Tech Stack

| Layer | Choice | Why |
|---|---|---|
| Portal | Next.js 14 (App Router) + Tailwind | Fast SSR, stakeholder UX |
| Services (platform) | Go (gin) | Small, fast, low memory |
| Services (agents/LLM) | Python (FastAPI) | Best LLM/tooling ecosystem |
| Orchestration | Temporal.io | Durable workflows, retries, state |
| Event bus | NATS JetStream | Simple, fast, durable |
| LLM abstraction | LiteLLM | Swap OpenAI/Anthropic/local |
| RAG | Qdrant + LangChain | Vector search over codebase |
| DB | PostgreSQL (per service) | Reliable, familiar |
| Cache/queue | Redis | Sessions, rate limits |
| Sandbox | Docker-in-Docker workspaces | Isolated agent execution |
| Secrets | Vault | Credentials per project |
| Observability | OpenTelemetry + Grafana/Loki/Tempo | End-to-end agent traces |
| CI for agents | Built by `agent-deploy` itself | Bootstrapping |
| Containers | Docker + Compose (dev), Swarm/k8s (prod) | Requirement |

---

## 8. Repository Layout (monorepo)

```
aisdlc/
├── docker-compose.yml            # full stack, one command
├── docker-compose.dev.yml        # hot-reload overrides
├── .env.example
├── PLAN.md  ARCHITECTURE.md  ROADMAP.md  AGENTS.md  ops/
├── shared/
│   ├── proto/                    # event schemas (JSON Schema / protobuf)
│   └── sdk-py  sdk-go  sdk-ts/   # clients per language
├── services/
│   ├── gateway/  identity/  portal/  backlog/  orchestrator/
│   ├── vcs/  llm-gateway/  knowledge/  sandbox/  notifications/
│   ├── secrets/  observability/
│   └── agents/
│       ├── triage/  requirements/  architect/  developer/
│       ├── reviewer/  qa/  deploy/  monitor/
└── infra/
    ├── postgres/  nats/  qdrant/  vault/  otel/
    └── k8s/                       # prod manifests (later)
```

---

## 9. Docker Layout

- **Base image** `aisdlc/base` = slim Python/Node/Go + OTel agent + `shared/sdk`.
- Every service has its own `Dockerfile` (multi-stage, distroless where possible).
- `docker-compose.yml` wires: infra (postgres, nats, qdrant, vault, redis,
  otel-collector, grafana) + platform services + agent services + portal.
- `sandbox` service talks to the host Docker socket (mounted read-only) to spin
  up ephemeral dev containers for `agent-developer` / `agent-qa`.
- Healthchecks + dependency ordering via compose `depends_on: condition`.
- One command: `make up` → `docker compose up -d` → portal at http://localhost:3000.

---

## 10. Security & Safety

- Agents never run on the host; all code execution in ephemeral sandboxes, no
  network egress except allow-listed (git, registry, llm-gateway).
- Secrets injected via Vault, never in env of agent containers.
- Every LLM call + tool call logged with `trace_id` → full audit replay.
- Each agent has an allow-list of tools (developer can git+shell; reviewer can
  read+comment only).
- Resource quotas per agent run (CPU, time, tokens, cost ceiling).
- Mandatory human gates at merge + prod (configurable to auto for low-risk).

---

## 11. Roadmap (phased)

See **ROADMAP.md** for milestone breakdown. Summary:
- **M0 Bootstrap** (this repo, compose, infra)
- **M1 Foundation** (identity, backlog, portal, orchestrator skeleton)
- **M2 First agent loop** (triage → requirements → human approval, end-to-end)
- **M3 Code agents** (developer + reviewer + qa + sandbox, real PR)
- **M4 Release** (deploy agent, staging→prod, monitoring)
- **M5 Self-improvement** (agent-monitor feedback loop, RAG tuning)

---

## 12. Open Decisions

- [ ] LLM provider default (OpenAI vs Anthropic vs local Ollama) — make configurable.
- [ ] Orchestration: Temporal vs in-house FSM on NATS — lean Temporal.
- [ ] Single monorepo vs polyrepo — start monorepo.
- [ ] Prod runtime: Docker Swarm vs Kubernetes — Swarm first, k8s later.
- [ ] Pricing/cost guardrails per project.
```

Next: I'll add the architecture deep-dive, the roadmap, and the per-agent spec.