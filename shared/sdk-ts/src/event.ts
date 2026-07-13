/**
 * Canonical event envelope on the NATS bus (ARCHITECTURE.md §6).
 * M0 skeleton: types + helpers are stable; the NATS transport lands in M1.
 */

export const ENVELOPE_VERSION = 1;

export interface Envelope<P = unknown> {
  id: string;            // UUID; consumers dedupe on this
  stream: string;        // JetStream stream name
  type: string;          // e.g. "task.finished"
  ts: string;            // ISO-8601 (UTC)
  trace_id?: string;     // OpenTelemetry trace id
  subject: string;       // NATS subject, e.g. "task:1234"
  payload: P;            // type-specific (see shared/proto/events.json)
  version: number;       // envelope schema version
}

export class EnvelopeError extends Error {}

/** Validate the version-independent invariants of an envelope. */
export function validateEnvelope(env: Envelope): void {
  if (!env.id) throw new EnvelopeError("envelope.id is required");
  if (!env.stream || !env.type || !env.subject) {
    throw new EnvelopeError("envelope stream/type/subject are required");
  }
  if (env.version !== ENVELOPE_VERSION) {
    throw new EnvelopeError(
      `envelope version mismatch: ${env.version} != ${ENVELOPE_VERSION}`,
    );
  }
  if (env.payload === undefined || env.payload === null) {
    throw new EnvelopeError("envelope.payload is required");
  }
}

/** Build a validated envelope with a generated id + timestamp. */
export function newEnvelope<P>(
  stream: string,
  type: string,
  subject: string,
  payload: P,
  trace_id?: string,
): Envelope<P> {
  const env: Envelope<P> = {
    id: randomUuid(),
    stream,
    type,
    subject,
    payload,
    ts: new Date().toISOString(),
    trace_id,
    version: ENVELOPE_VERSION,
  };
  validateEnvelope(env);
  return env;
}

function randomUuid(): string {
  const g = globalThis as { crypto?: { randomUUID?: () => string } };
  if (g.crypto?.randomUUID) return g.crypto.randomUUID(); // Node 16.7+/browsers
  // RFC 4122 v4 fallback for old runtimes.
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    return (c === "x" ? r : (r & 0x3) | 0x8).toString(16);
  });
}
