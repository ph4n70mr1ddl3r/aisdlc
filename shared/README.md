# shared/ — cross-service contracts & SDKs

Everything every service agrees on lives here, so the model-driven core and the
AI workforce speak the same language. See [PLAN.md §8](../PLAN.md).

| Package | What it is |
|---|---|
| [`proto/`](./proto) | The canonical event contract: the envelope JSON Schema + the event/stream catalog. **Source of truth.** |
| [`sdk-go/`](./sdk-go) | Go types + bus abstraction (metadata-api, data-api, gateway, …) |
| [`sdk-py/`](./sdk-py) | Python types + bus abstraction (workflow/rules/llm-gateway/agent-runtime, …) |
| [`sdk-ts/`](./sdk-ts) | TypeScript types + bus abstraction (portal, …) |

## Status (M0)
Skeletons. The **event envelope** types and the **idempotent consumer** helper
are stable and tested. The NATS JetStream transport and OpenTelemetry trace
propagation land with the first real consumer (M1+).

## Validate
```bash
make test   # parses the proto catalog + compile-checks the SDKs
```
