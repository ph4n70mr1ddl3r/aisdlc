"""Bus abstraction + idempotent consumer helper (ARCHITECTURE.md §6)."""
from __future__ import annotations

import threading
from typing import Any, Callable, Protocol

from .event import Envelope


class Handler(Protocol):
    def __call__(self, env: Envelope) -> None: ...


class DedupeStore(Protocol):
    """Records envelope IDs; used by Idempotent to skip JetStream redeliveries."""

    def seen(self, env_id: str) -> bool:
        """True if env_id was already recorded, else record it and return False."""


class MemoryStore:
    """In-memory DedupeStore for tests/dev only (not durable)."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._seen: set[str] = set()

    def seen(self, env_id: str) -> bool:
        with self._lock:
            if env_id in self._seen:
                return True
            self._seen.add(env_id)
            return False


def Idempotent(store: DedupeStore, handler: Callable[[Envelope], None]) -> Handler:
    """Wrap a handler so duplicate envelope IDs are processed exactly once."""

    def wrapped(env: Envelope) -> None:
        if store.seen(env.id):
            return
        handler(env)

    return wrapped


class Bus(Protocol):
    """Publish/subscribe to event envelopes.

    The M0 skeleton ships an in-process implementation for tests; the NATS
    JetStream implementation (extras: ``aisdlc[nats,otel]``) arrives in M1.
    """

    def publish(self, env: Envelope) -> None: ...
    def subscribe(self, stream: str, subject: str, handler: Handler) -> Any: ...


class MemoryBus:
    """In-process Bus for unit tests. Not for production."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._subs: list[tuple[str, str, Handler]] = []

    def publish(self, env: Envelope) -> None:
        env.validate()
        with self._lock:
            subs = list(self._subs)
        for stream, pattern, handler in subs:
            if stream == env.stream and _match(pattern, env.subject):
                handler(env)

    def subscribe(self, stream: str, subject: str, handler: Handler) -> Any:
        if not stream or handler is None:
            raise ValueError("stream and handler are required")
        entry = (stream, subject, handler)
        with self._lock:
            self._subs.append(entry)

        def close() -> None:
            with self._lock:
                try:
                    self._subs.remove(entry)
                except ValueError:
                    pass

        return close


def _match(pattern: str, subject: str) -> bool:
    if not pattern or pattern == ">":
        return True
    if pattern == subject:
        return True
    if pattern.endswith(".*"):
        prefix = pattern[:-1]
        return subject.startswith(prefix) and "." not in subject[len(prefix):]
    return False
