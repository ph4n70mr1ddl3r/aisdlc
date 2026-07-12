# Agentic SDLC Platform

An **AI-run IT department delivered as a service**. Stakeholders log in,
request anything — a feature, a bug fix, support, a whole new application — and
an **autonomous AI workforce** wearing many hats runs the entire lifecycle:
intake, design, build, ship, support, monitor. It's **model-driven** (users,
workflows, UIs, rules, roles — even the AI org chart — are metadata in the DB),
so **the platform builds and extends itself**.

**Microservices. Docker-first. One `docker compose up`.**

---

## 📚 Read this in order

| Doc | What's inside |
|---|---|
| **[PLAN.md](./PLAN.md)** | Vision, the four pillars, architecture, services catalog, the self-build loop |
| **[METADATA.md](./METADATA.md)** | ⭐ The data dictionary — entities, fields, UIs, workflows, rules, roles, personas, all as metadata |
| **[PERSONAS.md](./PERSONAS.md)** | The AI org chart; workers vs personas (hats); RACI by metadata; multi-hat runtime |
| **[LIFECYCLE.md](./LIFECYCLE.md)** | Request → project → delivery (Track A metadata / Track B code) → support loop |
| **[ARCHITECTURE.md](./ARCHITECTURE.md)** | Model-driven core engines, generic data/UI runtime, agent runtime, control flow |
| **[ROADMAP.md](./ROADMAP.md)** | Milestones M0→M7 |
| **[SETUP.md](./SETUP.md)** | Local dev: prerequisites, `make` targets, port map |

---

## TL;DR

```
Stakeholder ──"we need an Asset app"──► Request
                                          │ (L1 Support triages)
                            ┌─────────────▼──────────────┐
                            │      Orchestrator          │ reads RACI template
                            └─────────────┬──────────────┘
            PM → BA → Architect → Metadata Engineer → Test → Release
                          (personas assigned from the agent pool)
                                          │
                          AUTHOR METADATA → ddl-engine → PUBLISH
                                          │
              Asset Management app appears in the portal INSTANTLY:
              CRUD, workflows, dashboards, roles. No code written.
                                          │
                          SRE monitors → Support helps → new requests → (loop)
```

- **~19 services**: model-driven core (7 engines) + platform (12) + workforce (2).
- **Two delivery tracks**: Track A = metadata (instant, ~80% of requests);
  Track B = real code via sandbox+VCS (full SDLC, ~20%).
- **Multi-hat agents**: a worker pool takes on personas (PM, Architect, Dev,
  QA, SRE, Support…). The org chart is metadata and can reorganize itself.
- **Self-extending**: agents author metadata → publish → live. The platform
  grows its own capabilities.

## Quick start (once M0 lands)
```bash
cp .env.example .env
docker compose up -d
open http://localhost:3000     # portal (renders from metadata)
open http://localhost:3001     # grafana
```

## Status
Planning complete. Implementation begins at **M0 — Bootstrap** then **M1 —
Model-Driven Core** (the foundation everything else builds on). See
[ROADMAP.md](./ROADMAP.md). Estimated ~8–10 weeks to a platform that builds
both metadata and code apps autonomously.
