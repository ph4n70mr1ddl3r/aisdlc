# Agent Specs — Agentic SDLC Platform

One file per agent role. Each agent is a thin service implementing the runtime
contract from [ARCHITECTURE.md §3](./ARCHITECTURE.md). The "intelligence" is
the system prompt + tool set + RAG grounding; the service just runs the ReAct
loop against `llm-gateway`.

Common fields for every agent:

| Field | Description |
|---|---|
| Trigger event | Event that causes orchestrator to dispatch this agent |
| Inputs | Payload it receives |
| Outputs / artifacts | What it returns + emits |
| Tools | Capability-scoped tool allow-list |
| Human gate after | Whether a human must approve before next phase |
| Success criteria | Definition of done |
| Failure handling | Retry / escalate behavior |

---

## agent-triage — Product Owner
- **Trigger:** `ticket.created`
- **Inputs:** ticket {title, description, reporter}
- **Outputs:** ticket meta {type, severity, priority, epic, duplicates[]}, event `ticket.triaged`
- **Tools:** `readTicket`, `searchBacklog` (semantic dup check), `setMeta`
- **Gate after:** none (auto)
- **Done when:** type+severity+priority set; duplicate candidates noted
- **Fail:** default severity MEDIUM, route to maintainer

## agent-requirements — Business Analyst
- **Trigger:** `ticket.triaged`
- **Inputs:** ticket + reporter history
- **Outputs:** `Spec` {user stories, acceptance criteria, scope, open_questions[]}, event `spec.drafted`
- **Tools:** `readTicket`, `askHuman` (portal Q&A), `writeSpec`, `searchDocs`
- **Gate after:** **SPEC_REVIEW (human)** — stakeholder approves/edits spec
- **Done when:** spec has ≥1 acceptance criterion and 0 unresolved questions (or explicitly deferred)
- **Fail:** after 2 rounds of clarifying Qs unanswered, escalate to maintainer

## agent-architect — Solution Architect
- **Trigger:** `spec.approved`
- **Inputs:** spec + repo snapshot (via RAG)
- **Outputs:** `Design` {approach, file plan, API contracts (OpenAPI), data model changes, risks, test plan}
- **Tools:** `readSpec`, `readCode` (RAG), `writeDesign`, `drawDiagram` (mermaid)
- **Gate after:** ARCH_REVIEW (auto unless config requires human)
- **Done when:** design covers all acceptance criteria + non-functional reqs
- **Fail:** if spec implies > budget or cross-cutting risk → request maintainer

## agent-developer — Engineer
- **Trigger:** `design.approved`
- **Inputs:** design + repo ref
- **Outputs:** branch `ai/ticket-<id>`, commits, opened `ChangeRequest`/PR
- **Tools:** `gitClone`, `openBranch`, `editFiles`, `runShell(sandbox)`, `commit`, `openPR`
- **Gate after:** none (goes to reviewer)
- **Done when:** acceptance criteria implementable, local build+lint green
- **Fail:** if blocked > N attempts → comment on ticket, return to architect

## agent-reviewer — Code Reviewer + Security
- **Trigger:** `change.pr_opened`
- **Inputs:** PR diff + design + tests
- **Outputs:** comments, `approvePR` / `requestChanges`, SAST + dep-audit report
- **Tools:** `readPR`, `runTests`, `runSAST`, `commentPR`, `approvePR`, `requestChanges`
- **Gate after:** **PR_MERGE (human)** unless project allows auto-merge
- **Done when:** 0 blocking comments and tests + SAST green
- **Fail:** request changes → loops back to developer (max 3 iterations, then human)

## agent-qa — QA / SDET
- **Trigger:** developer commits (can run parallel to reviewer)
- **Inputs:** PR ref + acceptance criteria
- **Outputs:** generated tests, coverage report, test results
- **Tools:** `gitClone`, `writeTests`, `runTests`, `reportCoverage`
- **Gate after:** none
- **Done when:** all acceptance criteria have passing tests, coverage ≥ threshold
- **Fail:** if tests can't pass → comment, loop to developer

## agent-deploy — DevOps / Release Engineer
- **Trigger:** `pr.merged` (after UAT approval)
- **Inputs:** merged ref, env config, secrets ref
- **Outputs:** image tag, `Build` record, deployed env URL, healthcheck result
- **Tools:** `buildImage`, `pushImage`, `deployTo(env)`, `healthCheck`, `rollback`
- **Gate after:** **PROD_DEPLOY (human)** between staging and prod
- **Done when:** staging healthy; prod healthy post-approval; release tagged
- **Fail:** healthcheck fail → auto-rollback, file incident ticket

## agent-monitor — SRE
- **Trigger:** continuous (event subscription), not ticket-driven
- **Inputs:** metrics, logs, deploy events
- **Outputs:** incident tickets on SLO breach, auto-rollback on hard failure, post-deploy health summary
- **Tools:** `queryMetrics`, `queryLogs`, `fileIncident`, `triggerRollback`, `notifyOnCall`
- **Gate after:** none (closes the loop into the backlog)
- **Done when:** steady state or incident filed
- **Fail:** on ambiguous signal, alert on-call human

---

## Prompt engineering notes
- Each agent ships a versioned system prompt in `services/agents/<role>/prompts/`.
- Prompts emphasize: cite the spec/design, use tools, never invent files/APIs,
  stop when acceptance criteria are met, report blockers honestly.
- Prompts + evals (golden tickets) live in repo and run in CI to catch regressions when models change.

## Cost & safety
- Every run has a budget envelope (tokens, USD, wall-clock) enforced by the orchestrator.
- Tool calls require a capability token scoped to the run; sandbox egress is allow-listed.
- All actions recorded with `trace_id` for replay and audit.
