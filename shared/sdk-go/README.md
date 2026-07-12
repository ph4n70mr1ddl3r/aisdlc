# sdk-go

Shared Go types + bus abstraction for the Agentic SDLC Platform. Stdlib-only at
M0 (no external deps, so it builds offline).

```go
import "github.com/ph4n70mr1ddl3r/aisdlc/shared/sdk-go"

bus := aisdlc.NewMemoryBus()
store := aisdlc.NewMemoryStore()

_ = bus.Subscribe(ctx, "tasks", "task:*", aisdlc.Idempotent(store, func(ctx context.Context, env aisdlc.Envelope) error {
    // ... handle env.Payload ...
    return nil
}))

payload, _ := json.Marshal(map[string]any{"task_id": "1234", "status": "done"})
_ = bus.Publish(ctx, aisdlc.Envelope{
    Stream: "tasks", Type: "task.finished", Subject: "task:1234",
    ID: uuid.NewString(), Payload: payload, Version: aisdlc.EnvelopeVersion,
})
```

## Status (M0)
Stable: `Envelope`, `Validate`, `Bus`, `Handler`, `Idempotent`, `MemoryBus`,
`MemoryStore`. Pending (M1): a NATS JetStream `Bus`, `DedupeStore` backed by
Redis, and OpenTelemetry trace propagation (`trace_id`).
