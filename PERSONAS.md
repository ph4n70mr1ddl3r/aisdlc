# AI Workforce & Personas — the org chart is metadata

> In this platform the "IT department" is staffed by AI. Agents are **generic
> workers**; each task assigns them a **persona** (a hat). One agent can be a
> Product Manager on project X and a QA Engineer on project Y. The departments,
> personas, reporting lines, and RACI rules are all **metadata** — so the
> workforce can reorganize and grow itself.

This file replaces the old fixed "8 phase-agents" model from AGENTS.md. The
runtime contract (ReAct loop, tools, budget) still applies — see
ARCHITECTURE.md.

---

## 1. Worker vs Persona (the key idea)

```
┌────────────┐  wears, per task   ┌──────────────┐
│   Agent    │ ◄─────────────────►│   Persona    │
│ (a worker) │                    │    (a hat)   │
│ pool row   │                    │ prompt+tools+KPIs
└────────────┘                    └──────────────┘
```

- **`agent`** = a scalable runtime process in the `agent-runtime` pool. It has
  capacity, a cost ledger, a status. By itself it has *no role*.
- **`persona`** = a role definition: system prompt, tool allow-list, model
  preference, KPIs, cost budget, department. Stored in the `personas` table.
- **`assignment`** = "agent A wears persona P for task T". On task start the
  orchestrator injects P's prompt + tools + budget into A. On finish, A is free.

This means we can scale workers (cheap, horizontal) independently of roles
(rich, evolving), and one worker covers many seats.

---

## 2. Departments (the AI org chart)

```
Office of the CTO
├── Product (PMO)
│   ├── Product Management
│   ├── Business Analysis
│   └── UX/Service Design
├── Engineering
│   ├── Solution Architecture
│   ├── Backend Engineering
│   ├── Frontend Engineering
│   ├── Data Engineering
│   └── Platform Engineering
├── Quality (QA)
│   ├── Test Engineering (SDET)
│   └── Release Management
├── Reliability (DevOps/SRE)
│   ├── DevOps
│   ├── SRE / On-call
│   └── Security
├── Service Desk (Support)
│   ├── L1 Support
│   ├── L2 Support
│   └── Knowledge Management
└── Governance
    ├── Compliance/Audit
    └── Vendor / Cost
```

Each department is a row in `departments`; its mission and headcount (min/max
concurrent personas) are metadata. The CTO persona prioritizes the portfolio
across departments.

---

## 3. Persona library (seed; extendable)

| Persona | Dept | Owns | Tools (capability-scoped) |
|---|---|---|---|
| **CTO / Portfolio** | Product | Prioritization, resourcing, trade-offs | readRequests, setPriority, assignBudget |
| **Product Manager** | Product | Scope, acceptance criteria, roadmap | readRequest, writeSpec, askHuman, searchDocs |
| **Business Analyst** | Product | Requirements, process maps | readRequest, writeSpec, searchDocs, askHuman |
| **UX/Service Designer** | Product | Flows, view layouts, menus | readSpec, writeView, writeMenu, drawDiagram |
| **Solution Architect** | Engineering | Tech design, entity/workflow design, risk | readSpec, readCode(RAG), writeDesign, writeEntity, writeWorkflow |
| **Backend Engineer** | Engineering | Logic, APIs, integrations (Track B) | gitClone, editFiles, runShell, openPR |
| **Frontend Engineer** | Engineering | Custom widgets (Track B) | gitClone, editFiles, openPR |
| **Metadata Engineer** | Engineering | Authors entities/fields/views/rules (Track A) | writeEntity, writeField, writeView, writeRule, requestMigration |
| **Data Engineer** | Data | Pipelines, migrations, RAG ingest | runSQL, writePipeline, embedDocs |
| **Platform Engineer** | Engineering | Core service changes (Track B) | gitClone, editFiles, runShell, openPR |
| **Test Engineer (SDET)** | QA | Tests, coverage, UAT scripts | writeTests, runTests, reportCoverage |
| **Release Manager** | QA | Build, env promotion, change records | buildImage, deployTo, createChange |
| **DevOps** | Reliability | Infra, pipelines, secrets | applyTerraform, deployTo, rotateSecret |
| **SRE / On-call** | Reliability | Incidents, rollback, SLOs | queryMetrics, queryLogs, rollback, fileIncident |
| **Security Analyst** | Reliability | Threat modeling, SAST, review | runSAST, auditSecrets, writeThreatModel |
| **L1 Support** | Service Desk | Triage requests/incidents, FAQ | readRequest, classify, reply, searchKnowledge |
| **L2 Support** | Service Desk | Diagnose, reproduce, escalate | readRequest, runRepro, escalate, writeKnowledge |
| **Knowledge Manager** | Service Desk | Articles, runbooks | writeKnowledge, indexDocs |
| **Compliance/Audit** | Governance | Policy, audit trails, data classification | readAudit, writePolicy, classifyData |
| **Scrum Master** | Product | Cadence, blockers, reporting | manageSprint, reportBurndown, unblock |

Each persona ships as: `personas/<name>/{prompt.md, tools.yaml, kpis.yaml, model.yaml}`.
Seeded into the DB at bootstrap; editable as metadata; version-controlled.

---

## 4. How tasks get staffed (RACI by metadata)

For each `request_type` there's a `raci_template` mapping phases → personas as
**R**esponsible / **A**ccountable / **C**onsulted / **I**nformed.

Example — `feature` request:
| Phase | R | A | C | I |
|---|---|---|---|---|
| Intake/triage | L1 Support | Product Manager | — | requester |
| Requirements | Business Analyst | Product Manager | UX, Architect | requester |
| Design | Solution Architect | Product Manager | Security, Data | — |
| Build (Track A) | Metadata Engineer | Solution Architect | Backend, Frontend | — |
| Build (Track B) | Backend/Frontend Engineer | Solution Architect | Platform | — |
| Test | Test Engineer | Release Manager | — | — |
| Review/Approve | Security Analyst | Solution Architect | Compliance | requester |
| Deploy | Release Manager | DevOps | SRE | requester |
| Support | L2 Support | SRE | — | requester |

The orchestrator reads this template, creates `tasks`, and assigns personas to
free agents from the pool — honoring department capacity caps and tenant budgets.

---

## 5. Persona runtime injection

When an agent picks up a task, `agent-runtime` receives:
```json
{ "task_id": 1234, "persona_id": "solution_architect",
  "context": { "request_id": 42, "spec": {...}, "rag_hits": [...] },
  "tools": ["readSpec","readCode","writeDesign","writeEntity","writeWorkflow"],
  "model": "claude-3-5-sonnet", "budget": { "tokens": 200000, "usd": 5, "minutes": 20 },
  "kpis": { "criteria_coverage": 1.0, "risk_notes_present": true } }
```
It runs the ReAct loop with **only** those tools, logs to `agent_runs`, and
emits `task.finished` with artifacts. KPIs become acceptance checks; failing
them loops the task back.

---

## 6. Growing & reorganizing the workforce

Because it's all metadata:
- **Add a role**: CTO persona drafts a new persona row + prompt; Governance
  approves; live. (The org hires itself.)
- **Reorganize**: move a department, change reporting — update `departments`.
- **Right-size**: `workforce-api` observes utilization; CTO persona proposes
  capacity changes (spin up/down agents, switch models for cost).
- **Learn**: closed tasks feed RAG + persona prompt evals; personas improve
  over versions.

---

## 7. Human-in-the-loop (still first-class)

Personas flag for humans at gates (default on, risk-tunable):
spec sign-off, design review, change approval (ITIL), UAT, prod deploy, and any
action touching **secrets/prod-data/external systems**. Approvals are records
in metadata-defined entities, surfaced in the portal and via notifications.
