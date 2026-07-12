"""Canonical event envelope (ARCHITECTURE.md §6)."""
from __future__ import annotations

import json
import uuid
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any

ENVELOPE_VERSION = 1


class EnvelopeError(ValueError):
    """Raised when an envelope fails validation."""


@dataclass
class Envelope:
    """The canonical event wrapper published on the NATS bus."""

    stream: str
    type: str
    subject: str
    payload: dict[str, Any]
    id: str = field(default_factory=lambda: str(uuid.uuid4()))
    ts: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    trace_id: str | None = None
    version: int = ENVELOPE_VERSION

    def validate(self) -> None:
        if not self.id:
            raise EnvelopeError("envelope.id is required")
        if not (self.stream and self.type and self.subject):
            raise EnvelopeError("envelope stream/type/subject are required")
        if self.version != ENVELOPE_VERSION:
            raise EnvelopeError(
                f"envelope version mismatch: {self.version} != {ENVELOPE_VERSION}"
            )
        if self.payload is None:
            raise EnvelopeError("envelope.payload is required")

    def to_json(self) -> str:
        self.validate()
        return json.dumps(
            {
                "id": self.id,
                "stream": self.stream,
                "type": self.type,
                "ts": self.ts.isoformat(),
                "trace_id": self.trace_id,
                "subject": self.subject,
                "payload": self.payload,
                "version": self.version,
            },
            sort_keys=True,
        )

    @classmethod
    def from_json(cls, data: str | bytes) -> "Envelope":
        obj = json.loads(data)
        try:
            return cls(
                stream=obj["stream"],
                type=obj["type"],
                subject=obj["subject"],
                payload=obj["payload"],
                id=obj["id"],
                ts=datetime.fromisoformat(obj["ts"]),
                trace_id=obj.get("trace_id"),
                version=obj.get("version", ENVELOPE_VERSION),
            )
        except KeyError as exc:  # pragma: no cover - defensive
            raise EnvelopeError(f"envelope missing field: {exc}") from exc
