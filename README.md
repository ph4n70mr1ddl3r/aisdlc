# Agentic SDLC Platform

An **AI-managed software development lifecycle**: stakeholders sign in, file
tickets, and an autonomous multi-agent system runs the whole cycle —
requirements → design → code → review → test → release → monitor — looping back
into the backlog. Humans approve at gates; AI does the rest.

**Microservices. Docker-first. One `docker compose up`.**

---

## 📚 Read this in order

| Doc | What's inside |
|---|---|
| **[PLAN.md](./PLAN.md)** | The master plan: vision, goals, architecture, services catalog, roadmap pointer |
| **[ARCHITECTURE.md](./ARCHITECTURE.md)** | Control flow, event schema, agent runtime contract, sandbox, RAG, observability |
| **[AGENTS.md](./AGENTS.md)** | Per-agent specs: trigger, inputs/outputs, tools, gates, success criteria |
| **[ROADMAP.md](./ROADMAP.md)** | Milestones M0→M5 with demoable checkpoints |
| **[SETUP.md](./SETUP.md)** | Local dev: prerequisites, `make` targets, port map |

---

## TL;DR

```
┌─────────────┐   file ticket   ┌───────────┐
│  Stakeholder│ ───────────────► │  Backlog  │
│   Portal    │ ◄── approvals ──└─────┬─────┘
└─────────────┘                      │
                            ┌────────▼─────────┐
                            │   Orchestrator   │  ← ticket state machine
                            └────────┬─────────┘
        ┌──────────┬──────────┬──────┴───────┬──────────┬──────────┐
        ▼          ▼          ▼              ▼          ▼          ▼
    Triage   Requirements  Architect     Developer   Reviewer      QA
                                                                  │
                            … → Deploy → Monitor → (new tickets) ─┘
```

**19 services** (12 platform + 8 agent roles), each its own container, talking
over **NATS**, durable workflows on **Temporal**, per-service **Postgres**,
RAG via **Qdrant**, LLM access via **LiteLLM**, all under **OpenTelemetry**.

## Quick start (once M0 lands)
```bash
cp .env.example .env
docker compose up -d
open http://localhost:3000     # portal
open http://localhost:3001     # grafana
```

## Status
Planning complete. Implementation begins at **M0 — Bootstrap**
(see [ROADMAP.md](./ROADMAP.md)). Estimated ~6–8 weeks to a full end-to-end
vertical slice (file ticket → AI ships to prod → monitors itself).
