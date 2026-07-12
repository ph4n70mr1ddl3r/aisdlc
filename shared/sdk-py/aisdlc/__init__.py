"""Shared SDK for the Agentic SDLC Platform.

See shared/proto/README.md for the event contract. This package is an M0
skeleton: the types and the idempotent handler are stable; the NATS JetStream
transport lands with the first real consumer (M1+). Stdlib-only at import time.
"""
from .event import ENVELOPE_VERSION, Envelope, EnvelopeError
from .bus import Bus, DedupeStore, Handler, Idempotent, MemoryBus, MemoryStore

__version__ = "0.0.1"

__all__ = [
    "ENVELOPE_VERSION",
    "Envelope",
    "EnvelopeError",
    "Bus",
    "DedupeStore",
    "Handler",
    "Idempotent",
    "MemoryBus",
    "MemoryStore",
]
