# proto/ — the event contract

The single source of truth for everything that crosses the NATS JetStream bus.
Every event is a JSON document wrapped by the **Envelope**.

## Files
- [`envelope.schema.json`](./envelope.schema.json) — the canonical wrapper (ARCHITECTURE.md §6).
- [`events.json`](./events.json) — the catalog: streams + event types + per-type payload schemas.

## The envelope
```json
{ "id":"uuid","stream":"tasks","type":"task.finished",
  "ts":"2026-07-12T10:00:00Z","trace_id":"...","subject":"task:1234",
  "payload":{...},"version":1 }
```
- `id` — UUID. **Consumers dedupe on this** (JetStream redelivers; every consumer must be idempotent).
- `subject` — NATS subject carrying the entity id (`task:1234`). **Consumers order on this.**
- `trace_id` — OpenTelemetry trace id for cross-service spans.
- `version` — envelope schema version (currently 1).

## Adding an event
1. Add an entry to `events` in `events.json` (`stream`, `subject` template, `payload` JSON Schema).
2. If the stream is new, add it to `streams`.
3. Refresh the typed SDK models (TODO: codegen in M1).

## Validation
```bash
make test   # parses the catalog + compile-checks the SDKs
```
Full JSON-Schema meta-validation (needs the `jsonschema` package):
```bash
python3 -m pip install jsonschema >/dev/null && python3 - <<'PY'
import json, jsonschema
jsonschema.Draft202012Validator.check_schema(json.load(open('envelope.schema.json')))
cat = json.load(open('events.json'))
for name, e in cat['events'].items():
    jsonschema.Draft202012Validator.check_schema(e['payload'])
print(f"catalog OK ({len(cat['events'])} events, {len(cat['streams'])} streams)")
PY
```
