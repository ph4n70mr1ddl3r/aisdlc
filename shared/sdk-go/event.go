// Package aisdlc provides the shared event-envelope type and a minimal bus
// abstraction for the Agentic SDLC Platform. See shared/proto/README.md.
//
// This is an M0 skeleton: the NATS JetStream transport and OpenTelemetry
// propagation land with the first real consumer (M1+). The Envelope type and
// the Idempotent handler helper are stable. The package is stdlib-only.
package aisdlc

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// EnvelopeVersion is the canonical event-envelope schema version.
const EnvelopeVersion = 1

// Envelope is the canonical event wrapper published on the NATS bus
// (ARCHITECTURE.md §6). time.Time marshals to RFC 3339 by default.
type Envelope struct {
	ID      string          `json:"id"`                 // UUID; consumers dedupe on this
	Stream  string          `json:"stream"`             // JetStream stream name
	Type    string          `json:"type"`               // e.g. "task.finished"
	TS      time.Time       `json:"ts"`                 // when produced (UTC)
	TraceID string          `json:"trace_id,omitempty"` // OpenTelemetry trace id
	Subject string          `json:"subject"`            // NATS subject, e.g. "task:1234"
	Payload json.RawMessage `json:"payload"`            // type-specific; see events.json
	Version int             `json:"version"`            // envelope schema version
}

// Validate checks the envelope's version-independent invariants.
func (e Envelope) Validate() error {
	if e.ID == "" {
		return errors.New("aisdlc: envelope.id is required")
	}
	if e.Stream == "" || e.Type == "" || e.Subject == "" {
		return errors.New("aisdlc: envelope stream/type/subject are required")
	}
	if e.Version < 1 {
		return fmt.Errorf("aisdlc: envelope version mismatch: got %d, minimum is 1", e.Version)
	}
	if len(e.Payload) == 0 || string(e.Payload) == "null" {
		return errors.New("aisdlc: envelope.payload is required")
	}
	return nil
}
