# AI Workforce & Personas (quick ref)

> The detailed model lives in **[PERSONAS.md](./PERSONAS.md)**. This file is the
> legacy index kept for the old "fixed phase-agents" framing — now superseded.

## What changed
Previously each SDLC phase was a dedicated service (`agent-triage`,
`agent-developer`, …). The new model is:

- **One** `agent-runtime` service running a **pool** of generic workers.
- A **persona library** (PM, Architect, Metadata Engineer, Backend Dev, QA,
  SRE, Support, …) stored as **metadata** in `personas`.
- Per task, the orchestrator **assigns a persona** (hat) to a worker:
  injects prompt + scoped tools + budget + KPIs.
- Agents can wear **multiple hats** across tasks; the org chart itself is
  metadata and can be reorganized/grown by the platform.

➡️ Read **PERSONAS.md** for the full department tree, persona library,
RACI-by-metadata, and runtime injection contract.

➡️ Read **LIFECYCLE.md** for how personas cooperate across the request→project→
delivery→support loop.
