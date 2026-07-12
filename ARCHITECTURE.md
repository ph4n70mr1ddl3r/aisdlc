# Architecture — Agentic SDLC Platform

Companion to [PLAN.md](./PLAN.md). This file goes deeper on contracts, control
flow, and the agent runtime.

---

## 1. Control Flow (one ticket, full cycle)

```
portal.fileTicket(desc)
   └─► POST /backlog/tickets ─► DB row (FILED)
       └─► emit ticket.created (NATS "tickets")
            └─► orchestrator.onTicketCreated:
                  start workflow(ticketId)  [Temporal]
                  └─► activity dispatch(agent="triage")
                       └─► POST agent-triage/run {ticketId}
                            └─► triage: LLM classify+prioritize
                            └─► PATCH ticket (status=TRIAGED, meta)
                       └─► activity dispatch("requirements")
                            └─► requirements agent: draft spec,
                                optionally portal.ask(reporterId, Qs)
                                (workflow pauses on human signal)
                       └─► wait event approval.decided (SPEC_REVIEW)
                       └─► dispatch("architect") → design doc
                       └─► ... developer → reviewer → qa → deploy ...
                       └─► ticket CLOSED; monitor agent subscribes.
```

The **orchestrator owns the state machine**; agents are stateless workers that
receive a `task` payload and return a `result` + side-effects (git commits,
PRs, comments). Pausing for human approval = Temporal `Signal`/`Timer`.

---

## 2. Event Schema (append-only bus)

Streams: `tickets`, `specs`, `designs`, `changes`, `builds`, `approvals`,
`agent.runs`, `incidents`.

Every event (canonical, JSON Schema-validated in `shared/proto`):
```json
{
  "id": "uuid",
  "stream": "tickets",
  "type": "ticket.created",
  "ts": "2026-07-12T18:00:00Z",
  "trace_id": "...",
  "subject": "ticket:42",
  "payload": { "ticket_id": 42, "project_id": 7, "reporter_id": 3 },
  "version": 1
}
```
Consumers are idempotent: dedupe on `event.id`, apply via `subject` ordering.

---

## 3. Agent Runtime Contract

All agent services implement the same interface:

```
POST /run
  body: { task_id, ticket_id, inputs, tools_allowed, deadline, budget }
  resp: 202 { run_id }
GET  /runs/{run_id} → { status: running|done|failed, result, artifacts, logs, tokens, cost }
POST /runs/{run_id}/cancel
```

Internally an agent run = ReAct loop:
```
system_prompt(role) + context(ticket, design, rag_hits) + toolset
  └─ loop until done or budget exhausted:
       LLM → thought + action(tool, args)
       execute tool (sandboxed) → observation
       append to trace
  └─ emit agent.run.finished event with artifacts[]
```

**Tool registry** (capability-scoped per agent):

| Agent | Tools |
|---|---|
| triage | readTicket, searchBacklog, setMeta |
| requirements | readTicket, askHuman, writeSpec, searchDocs |
| architect | readSpec, readCode(RAG), writeDesign, drawDiagram |
| developer | gitClone, openBranch, editFiles, runShell(sandbox), commit, openPR |
| reviewer | readPR, runTests, runSAST, commentPR, approvePR |
| qa | gitClone, writeTests, runTests, reportCoverage |
| deploy | buildImage, pushImage, deployTo(env), healthCheck, rollback |
| monitor | queryMetrics, queryLogs, fileIncident, triggerRollback |

Tools are implemented once in `shared/sdk-*` and authorized via a signed
capability token minted by the orchestrator per run.

---

## 4. Sandbox Architecture

`sandbox` service manages ephemeral dev environments for developer/qa/deploy:
```
agent-developer ──► sandbox.create({repo, branch, base_image})
                       └─► docker run --network sandbox-net --pids-limit ...
                            aisdlc/devbox:{repo} (git checkout, deps cached)
                  ◄── ws_url (exec over HTTP/WS)
agent-developer ──► sandbox.exec(ws_url, ["pytest"])
                  ──► sandbox.destroy(ws_url)
```
- No network egress except an allow-list proxy (registry, git, llm-gateway).
- CPU/mem/time quotas; auto-destroy on timeout.
- Ephemeral; source of truth stays in git.

---

## 5. RAG / Knowledge

`knowledge` service keeps an indexed view of the project so agents have context
without reading the whole repo:
- On every `pr.merged`, re-embed changed files into **Qdrant**.
- Index: code chunks (tree-sitter), docs, ADRs, past tickets+specs.
- Query API: `POST /search { q, k, filter: {path, lang} }`.
- Developer/architect/reviewer agents call it to ground decisions.

---

## 6. LLM Gateway

- Wraps providers (OpenAI, Anthropic, Bedrock, Ollama) behind one API.
- Per-tenant keys & quotas; cost metering per `agent.run`.
- Fallback chains (primary → fallback model).
- Streaming + tool-calling normalized.
- Prompt templates versioned in `services/llm-gateway/prompts/`.

---

## 7. Approvals (Human-in-the-loop)

The `approvals` sub-system inside `backlog` (+ `notifications`):
- Orchestrator creates `Approval(ref=spec:42, gate=SPEC_REVIEW)`.
- `notifications` emails stakeholder a signed one-click link + portal task.
- Decision → `approval.decided` event → orchestrator Signal resumes workflow.
- SLA timers; escalations; auto-reject on timeout (configurable).

---

## 8. Observability

Every agent action carries `trace_id`/`span_id` (OpenTelemetry).
- Collector → Tempo (traces), Loki (logs), Prometheus (metrics), Grafana (dashboards).
- Per-ticket "flight recorder" view in portal: timeline of phases, tokens,
  cost, diffs, test results.
- Audit log (append-only, hashed) of every privileged tool call for compliance.

---

## 9. Failure Modes & Mitigations

| Risk | Mitigation |
|---|---|
| Agent hallucinates API | Reviewer + tests gate; sandbox reproducible build |
| Infinite loops / runaway cost | Per-run budget (tokens/$/time) hard cap |
| Bad merge | Mandatory tests + reviewer gate; one-click rollback |
| Secret leak | Vault, no env in sandboxes, egress allow-list |
| Provider outage | LLM gateway fallback chain |
| Workflow stuck on approval | SLA timers + escalation to maintainer |
