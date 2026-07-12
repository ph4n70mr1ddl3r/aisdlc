# sdk-ts

Shared TypeScript types + bus abstraction for the Agentic SDLC Platform (portal,
widgets, …). M0 skeleton.

```ts
import {
  newEnvelope,
  idempotent,
  MemoryBus,
  MemoryStore,
} from "@aisdlc/sdk";

const bus = new MemoryBus();
const store = new MemoryStore();

bus.subscribe(
  "tasks",
  "task:*",
  idempotent(store, (env) => console.log(env.type, env.subject)),
);

bus.publish(
  newEnvelope("tasks", "task.finished", "task:1234", { status: "done" }),
);
```

## Status (M0)
Stable: `Envelope`, `validateEnvelope`, `newEnvelope`, `Bus`, `Handler`,
`idempotent`, `MemoryBus`, `MemoryStore`. Pending (M1): a NATS JetStream `Bus`
and OpenTelemetry trace propagation. Build/typecheck with `npm install && npm
run build` (needs `typescript`).
