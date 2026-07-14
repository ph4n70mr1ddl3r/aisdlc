package aisdlc

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEnvelopeValidate(t *testing.T) {
	now := time.Now().UTC()
	payload := json.RawMessage(`{"key":"val"}`)

	tests := []struct {
		name    string
		env     Envelope
		wantErr bool
	}{
		{
			name: "valid envelope",
			env: Envelope{
				ID:      "550e8400-e29b-41d4-a716-446655440000",
				Stream:  "test-stream",
				Type:    "test.event",
				TS:      now,
				Subject: "test:123",
				Payload: payload,
				Version: 1,
			},
			wantErr: false,
		},
		{
			name: "missing id",
			env: Envelope{
				Stream:  "test-stream",
				Type:    "test.event",
				TS:      now,
				Subject: "test:123",
				Payload: payload,
				Version: 1,
			},
			wantErr: true,
		},
		{
			name: "missing stream",
			env: Envelope{
				ID:      "550e8400-e29b-41d4-a716-446655440000",
				Type:    "test.event",
				TS:      now,
				Subject: "test:123",
				Payload: payload,
				Version: 1,
			},
			wantErr: true,
		},
		{
			name: "missing type",
			env: Envelope{
				ID:      "550e8400-e29b-41d4-a716-446655440000",
				Stream:  "test-stream",
				TS:      now,
				Subject: "test:123",
				Payload: payload,
				Version: 1,
			},
			wantErr: true,
		},
		{
			name: "missing subject",
			env: Envelope{
				ID:      "550e8400-e29b-41d4-a716-446655440000",
				Stream:  "test-stream",
				Type:    "test.event",
				TS:      now,
				Payload: payload,
				Version: 1,
			},
			wantErr: true,
		},
		{
			name: "invalid version",
			env: Envelope{
				ID:      "550e8400-e29b-41d4-a716-446655440000",
				Stream:  "test-stream",
				Type:    "test.event",
				TS:      now,
				Subject: "test:123",
				Payload: payload,
				Version: 0,
			},
			wantErr: true,
		},
		{
			name: "empty payload",
			env: Envelope{
				ID:      "550e8400-e29b-41d4-a716-446655440000",
				Stream:  "test-stream",
				Type:    "test.event",
				TS:      now,
				Subject: "test:123",
				Payload: nil,
				Version: 1,
			},
			wantErr: true,
		},
		{
			name: "null payload",
			env: Envelope{
				ID:      "550e8400-e29b-41d4-a716-446655440000",
				Stream:  "test-stream",
				Type:    "test.event",
				TS:      now,
				Subject: "test:123",
				Payload: json.RawMessage("null"),
				Version: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.env.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvelopeJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	original := Envelope{
		ID:      "550e8400-e29b-41d4-a716-446655440000",
		Stream:  "test-stream",
		Type:    "test.event",
		TS:      ts,
		TraceID: "trace-abc-123",
		Subject: "test:456",
		Payload: json.RawMessage(`{"hello":"world"}`),
		Version: 1,
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Envelope
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Stream != original.Stream {
		t.Errorf("Stream mismatch: got %q, want %q", decoded.Stream, original.Stream)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %q, want %q", decoded.Type, original.Type)
	}
	if !decoded.TS.Equal(original.TS) {
		t.Errorf("TS mismatch: got %v, want %v", decoded.TS, original.TS)
	}
	if decoded.TraceID != original.TraceID {
		t.Errorf("TraceID mismatch: got %q, want %q", decoded.TraceID, original.TraceID)
	}
	if decoded.Subject != original.Subject {
		t.Errorf("Subject mismatch: got %q, want %q", decoded.Subject, original.Subject)
	}
	if string(decoded.Payload) != string(original.Payload) {
		t.Errorf("Payload mismatch: got %s, want %s", decoded.Payload, original.Payload)
	}
	if decoded.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", decoded.Version, original.Version)
	}
}

func TestEnvelopeOmitEmptyTraceID(t *testing.T) {
	env := Envelope{
		ID:      "550e8400-e29b-41d4-a716-446655440000",
		Stream:  "test",
		Type:    "test.event",
		TS:      time.Now().UTC(),
		Subject: "test:1",
		Payload: json.RawMessage(`{}`),
		Version: 1,
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["trace_id"]; ok {
		t.Error("trace_id should be omitted when empty")
	}
}
