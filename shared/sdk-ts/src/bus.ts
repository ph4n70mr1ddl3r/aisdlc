/**
 * Bus abstraction + idempotent consumer helper (ARCHITECTURE.md §6).
 * M0 skeleton: in-memory implementation for tests; NATS transport in M1.
 */
import { validateEnvelope, type Envelope } from "./event.js";


export type Handler<P = unknown> = (env: Envelope<P>) => void | Promise<void>;

export interface DedupeStore {
  /** True if id was already recorded, else record it and return false. */
  seen(id: string): boolean;
}

export interface Subscription {
  close(): void;
}

export interface Bus {
  publish(env: Envelope): Promise<void>;
  subscribe(stream: string, subject: string, handler: Handler): Subscription;
}

export class MemoryStore implements DedupeStore {
  private seen_ = new Set<string>();
  seen(id: string): boolean {
    if (this.seen_.has(id)) return true;
    this.seen_.add(id);
    return false;
  }
}

/** Wrap a handler so duplicate envelope IDs are processed exactly once. */
export function idempotent<P>(
  store: DedupeStore,
  handler: Handler<P>,
): Handler<P> {
  return (env) => {
    if (store.seen(env.id)) return;
    return handler(env);
  };
}

/** In-process Bus for unit tests. Not for production. */
export class MemoryBus implements Bus {
  private subs: Array<{ stream: string; pattern: string; handler: Handler }> =
    [];

  async publish(env: Envelope): Promise<void> {
    validateEnvelope(env);
    for (const s of this.subs.slice()) {
      if (s.stream === env.stream && matchSubject(s.pattern, env.subject)) {
        await s.handler(env);
      }
    }
  }

  subscribe(stream: string, subject: string, handler: Handler): Subscription {
    const entry = { stream, pattern: subject, handler };
    this.subs.push(entry);
    const subs = this.subs;
    return {
      close() {
        const i = subs.indexOf(entry);
        if (i >= 0) subs.splice(i, 1);
      },
    };
  }
}

/** Minimal NATS-subset matcher: empty/">" = all, exact, or trailing ".*". */
function matchSubject(pattern: string, subject: string): boolean {
  if (!pattern || pattern === ">") return true;
  if (pattern === subject) return true;
  if (pattern.endsWith(".*")) return subject.startsWith(pattern.slice(0, -1));
  return false;
}
