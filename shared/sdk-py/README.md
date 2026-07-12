# sdk-py

Shared Python types + bus abstraction for the Agentic SDLC Platform. Stdlib-only
at import time at M0.

```python
from aisdlc import Envelope, Idempotent, MemoryBus, MemoryStore

bus = MemoryBus()
store = MemoryStore()

bus.subscribe("tasks", "task:*", Idempotent(store, lambda env: print(env.type, env.subject)))

bus.publish(Envelope(
    stream="tasks", type="task.finished", subject="task:1234",
    payload={"task_id": "1234", "status": "done"},
))
```

## Status (M0)
Stable: `Envelope` (+ `to_json`/`from_json`/`validate`), `Bus`, `Handler`,
`Idempotent`, `MemoryBus`, `MemoryStore`. Pending (M1): a NATS JetStream `Bus`
and a durable `DedupeStore` — install via `pip install "aisdlc[nats,otel]"` when
they land.
