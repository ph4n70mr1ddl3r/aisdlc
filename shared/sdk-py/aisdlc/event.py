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
        if self.version < 1:
            raise EnvelopeError(
                f"envelope version mismatch: {self.version} is below minimum (1)"
            )
        if self.payload is None:
            raise EnvelopeError("envelope.payload is required")

    def to_json(self) -> str:
        self.validate()
        d: dict[str, Any] = {
            "id": self.id,
            "stream": self.stream,
            "type": self.type,
            "ts": self.ts.isoformat(),
            "subject": self.subject,
            "payload": self.payload,
            "version": self.version,
        }
        if self.trace_id is not None:
            d["trace_id"] = self.trace_id
        return json.dumps(d, sort_keys=True)

    @classmethod
    def from_json(cls, data: str | bytes) -> "Envelope":
        try:
            obj = json.loads(data)
        except json.JSONDecodeError as exc:
            raise EnvelopeError(f"invalid envelope JSON: {exc}") from exc
        try:
            return cls(
                stream=obj["stream"],
                type=obj["type"],
                subject=obj["subject"],
                payload=obj["payload"],
                id=obj["id"],
                ts=datetime.fromisoformat(obj["ts"].replace("Z", "+00:00", 1)),
                trace_id=obj.get("trace_id"),
                version=obj.get("version", ENVELOPE_VERSION),
            )
        except KeyError as exc:  # pragma: no cover - defensive
            raise EnvelopeError(f"envelope missing field: {exc}") from exc
