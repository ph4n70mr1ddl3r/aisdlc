# Roadmap — Agentic SDLC Platform

Phased delivery. Each milestone is independently demoable and ends with a
working `docker compose up`.

Legend: ✅ done · 🔵 in progress · ⬜ next

---

## M0 — Bootstrap  ⬜  (~3 days)
Goal: a runnable empty platform.
- [ ] Repo scaffold (monorepo layout from PLAN §8)
- [ ] `docker-compose.yml` with infra only (postgres, nats, redis, qdrant,
      vault, otel-collector, grafana, loki, tempo, prometheus)
- [ ] `shared/proto` event JSON schemas + generators
- [ ] `shared/sdk-py` / `sdk-go` / `sdk-ts` skeletons (event bus client,
      capability tokens, OTel init)
- [ ] Base images `aisdlc/base-python`, `aisdlc/base-go`, `aisdlc/base-node`
- [ ] `make up/down/logs/test` developer workflow
**Demo:** `make up` → healthy Grafana + NATS, nothing else.

---

## M1 — Foundation  ⬜  (~1 week)
Goal: a stakeholder can sign up, log in, and file a ticket.
- [ ] `identity`: signup/login (JWT), roles, email verify (mock SMTP)
- [ ] `gateway`: Traefik routing, forwards JWT to services
- [ ] `backlog`: CRUD tickets, projects, emit `ticket.created`
- [ ] `portal`: Next.js — auth, ticket form, list view
- [ ] `notifications`: in-app + email (mailhog in dev)
- [ ] `orchestrator`: Temporal skeleton, subscribes to `ticket.created`,
      logs only (no agents yet)
**Demo:** Create account → file "make logo bigger" → see it in portal + NATS.

---

## M2 — First Agent Loop  ⬜  (~1 week)
Goal: triage + requirements agents, full loop with one human approval.
- [ ] `llm-gateway` (LiteLLM) with one provider key, cost metering
- [ ] `agent-triage`: classify/prioritize, update ticket meta
- [ ] `agent-requirements`: draft spec from ticket, optional clarifying Qs
- [ ] Approval flow for `SPEC_REVIEW` (email + portal)
- [ ] Orchestrator wires FILED → TRIAGED → SPEC_DRAFTING → SPEC_APPROVED
- [ ] Flight-recorder timeline in portal
**Demo:** File ticket → AI drafts spec → you approve in portal.

---

## M3 — Code Agents  ⬜  (~2 weeks)
Goal: AI opens a real PR against a sample repo.
- [ ] `vcs`: manage git repo (bare repos on volume), branch/PR API
- [ ] `sandbox`: ephemeral devboxes via Docker, allow-listed egress
- [ ] `knowledge`: Qdrant ingest of a sample repo (tree-sitter chunks)
- [ ] `agent-architect`: design doc from approved spec + RAG
- [ ] `agent-developer`: ReAct loop, edits files in sandbox, commits, opens PR
- [ ] `agent-qa`: generates + runs tests in sandbox, reports coverage
- [ ] `agent-reviewer`: reviews PR, runs SAST, comments / approves
- [ ] Orchestrator wires SPEC_APPROVED → ... → PR_READY (human merge gate)
**Demo:** Approve spec → AI designs, codes, tests, reviews → PR opens.

---

## M4 — Release  ⬜  (~1 week)
Goal: ship to staging, approve, ship to prod, then monitor.
- [ ] `agent-deploy`: build image, push to local registry, deploy to
      `staging` docker compose project, healthcheck
- [ ] UAT approval gate → `prod` deploy (blue/green or recreate)
- [ ] Rollback tooling (image tags, one click)
- [ ] `agent-monitor`: scrape Prom metrics, tail Loki, file incident tickets
      on SLO breach, auto-rollback on hard failure
- [ ] Secrets via Vault injected per env
**Demo:** Approve UAT → prod deploy → kill a service → monitor auto-files ticket.

---

## M5 — Self-Improvement & Polish  ⬜  (ongoing)
- [ ] Feedback loop: closed tickets feed RAG + prompt evals
- [ ] Per-project config of gates, models, budgets
- [ ] Multi-project, RBAC, audit export
- [ ] Cost dashboards per tenant
- [ ] Migrate prod runtime to Kubernetes (Helm chart) — optional

---

## Effort estimate (1–2 engineers)
~6–8 weeks to M4 (vertical slice, end-to-end single ticket). M5 is continuous.
