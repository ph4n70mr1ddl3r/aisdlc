# Request → Project → Delivery Lifecycle

> How anything a stakeholder asks for becomes a delivered, supported capability.
> One pipeline, two delivery tracks, one continuous improvement loop. This is
> the operational heartbeat of the AI IT department.

---

## 1. Intake (many doors, one queue)

A request can enter through:
- **Guided form** in the portal: pick type (feature/bug/support/infra/data/
  change/access), title, description, attachments.
- **Chat / NL**: "we need an asset management app" → L1 Support persona parses
  it into a structured `request`.
- **Email-to-ticket** gateway.
- **Auto-filed**: Monitor persona raises an incident; it spawns a bug request.
- **API** (integrations, webhooks).

All become rows in `requests`. Each `request_type` has a default workflow +
SLA + RACI template (metadata). One unified workboard.

```
                  ┌──── NL chat ────┐
   form ─────────►│                 │──► request (typed) ──► Orchestrator
   email ────────►│  L1 Support     │
   incident ─────►│  persona parses │
   api ──────────►└─────────────────┘
```

---

## 2. Triage & routing

1. **L1 Support** persona classifies: type, urgency, impact, duplicates,
   tenant, affected app/service.
2. **Product Manager** persona (or CTO for big items) decides: accept/reject,
   priority, target **project** (new or existing), method (scrum/kanban).
3. Orchestrator creates `project` + `epics`/`stories`/`tasks` per the RACI
   template; assigns personas to agents from the pool.

---

## 3. Decide the track (A or B)

The Architect persona, during design, chooses:

| | Track A — Metadata | Track B — Code |
|---|---|---|
| Fits when | New app/module, CRUD, workflows, rules, dashboards, roles | New platform service, custom widget, heavy integration, perf work |
| Output | Metadata rows (entities/fields/views/workflows/rules) | Real repo: services, tests, images |
| Deploy speed | seconds (publish) | minutes (build + CI/CD) |
| % of requests (typical) | ~80% | ~20% |

Most feature requests run Track A. Track B is invoked when the renderer/tooling
can't express the need, or when metadata itself needs extending.

---

## 4. Track A — Metadata delivery (the self-build)

```
Product Manager ──► scope + acceptance criteria
Business Analyst ──► requirements spec (stored as record)
UX/Service Designer ──► view layouts, menus, flows
Solution Architect ──► entity model, relationships, workflows, rules, roles
Metadata Engineer ──► AUTHORS the metadata (JSON) into metadata-api (draft)
   └─ ddl-engine ──► computes diff vs published ──► migration (up/down SQL)
Test Engineer ──► validates via generic portal renderer (no code yet)
Security Analyst ──► checks PII/permissions/data classification
   └─ PUBLISH (approval gate) ──► portal serves new app instantly
Release Manager ──► records the change; Support onboards users
```

No compiler touched. The capability is live the moment metadata is published.
Rollback = revert to the prior `metadata_bundle` version.

---

## 5. Track B — Code delivery (full agentic SDLC)

When real code is required (e.g. a new payment-integration service, a custom
3D asset viewer widget, or extending a core engine):

```
Solution Architect ──► design doc + API contracts
Backend/Frontend/Platform Engineer (personas) ──► branch, code in sandbox
Test Engineer (SDET) ──► generates + runs tests in sandbox
Security Analyst ──► SAST, dependency audit, threat model
Reviewer (Architect) ──► PR review ──► MERGE (approval gate)
Release Manager ──► build image ──► deploy dev→staging→prod (approvals)
SRE ──► healthchecks, SLOs, rollback ready
```
This is the original agentic SDLC, now expressed as **personas on Track B**.
Code lives in repos managed by `vcs`; deploys via `sandbox` + registry.

---

## 6. Gates & approvals (human-in-the-loop)

Gates are configurable per `request_type` and risk. Default gates:
1. **Spec sign-off** (stakeholder)
2. **Design review** (Architect or maintainer; auto for low-risk Track A)
3. **Change approval** (ITIL `change` record) — for anything touching prod
4. **UAT** (stakeholder on staging)
5. **Prod deploy** (release approval)
6. **Any action** touching secrets, prod data, or external systems → human

Approvals are records in metadata-defined entities; surfaced in portal + email.

---

## 7. Support & the continuous loop

After delivery:
- **Monitor (SRE) persona** watches metrics/logs/SLOs for the new capability.
- **L2 Support persona** handles user questions; writes `knowledge_articles`.
- Regressions or feature gaps → **auto-file a new request** → loops to §1.
- Closed work feeds **RAG + persona evals**, so the workforce improves.

```
   delivery ──► support ──► monitor ──► (incident/gap) ──► new request ──► ...
                              ▲                                              │
                              └──────────── continuous improvement ─────────┘
```

---

## 8. End-to-end example

> Stakeholder: "We need to track company laptops — who has what, and flag
> returns when someone leaves."

1. Filed as `feature` request. L1 Support classifies; PM accepts.
2. Project created. RACI staffs: BA, Architect, Metadata Engineer, Test, Release.
3. **Track A** chosen.
4. Architect designs: entities `Asset`, `AssetAssignment`, `Employee`(ref);
   fields; an `AssetLifecycle` workflow (Available→Assigned→Returning→Returned);
   a rule `on Employee.exit → transition assigned assets to Returning`; roles
   `AssetManager`, `Viewer`; list/form/detail views; a dashboard.
5. Metadata Engineer authors it; ddl-engine creates tables in tenant DB.
6. Test Engineer validates via the renderer; Security marks `Employee` PII.
7. **Publish** (stakeholder approves) → "Asset Management" app appears in the
   portal instantly. Users start tracking laptops.
8. SRE monitors; when an offboarding triggers the rule, it works; if a gap
   surfaces (e.g. "need depreciation field"), Support files a request → loop.

Total wall-clock for the bulk of it: minutes, not sprints. Code (Track B) was
never needed.
