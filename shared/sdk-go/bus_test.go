package aisdlc

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryStore_Seen(t *testing.T) {
	s := NewMemoryStore()

	seen, err := s.Seen("id-1")
	if err != nil {
		t.Fatalf("first Seen: %v", err)
	}
	if seen {
		t.Error("expected false for first call")
	}

	seen, err = s.Seen("id-1")
	if err != nil {
		t.Fatalf("second Seen: %v", err)
	}
	if !seen {
		t.Error("expected true for duplicate call")
	}

	// different id should still be new
	seen, err = s.Seen("id-2")
	if err != nil {
		t.Fatalf("Seen id-2: %v", err)
	}
	if seen {
		t.Error("expected false for new id")
	}
}

func TestMemoryStore_ConcurrentSafe(t *testing.T) {
	s := NewMemoryStore()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const goroutines = 10
	const idsPerGoroutine = 100

	done := make(chan struct{}, goroutines)
	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer func() { done <- struct{}{} }()
			for i := 0; i < idsPerGoroutine; i++ {
				id := string(rune(g*idsPerGoroutine + i))
				if _, err := s.Seen(id); err != nil {
					t.Errorf("Seen: %v", err)
					return
				}
			}
		}()
	}

	for g := 0; g < goroutines; g++ {
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatal("timeout waiting for goroutines")
		}
	}
}

func TestMemoryBus_PublishSubscribe(t *testing.T) {
	b := NewMemoryBus()
	ctx := context.Background()

	var received atomic.Int32
	sub, err := b.Subscribe(ctx, "test-stream", "test.*", func(_ context.Context, env Envelope) error {
		received.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	env := Envelope{
		ID:      "550e8400-e29b-41d4-a716-446655440000",
		Stream:  "test-stream",
		Type:    "test.event",
		TS:      time.Now().UTC(),
		Subject: "test:123",
		Payload: json.RawMessage(`{"key":"val"}`),
		Version: 1,
	}

	if err := b.Publish(ctx, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if got := received.Load(); got != 1 {
		t.Errorf("expected 1 delivery, got %d", got)
	}
}

func TestMemoryBus_NonMatchingSubject(t *testing.T) {
	b := NewMemoryBus()
	ctx := context.Background()

	var received atomic.Int32
	sub, err := b.Subscribe(ctx, "test-stream", "other.*", func(_ context.Context, env Envelope) error {
		received.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	env := Envelope{
		ID:      "550e8400-e29b-41d4-a716-446655440000",
		Stream:  "test-stream",
		Type:    "test.event",
		TS:      time.Now().UTC(),
		Subject: "test:123",
		Payload: json.RawMessage(`{}`),
		Version: 1,
	}

	if err := b.Publish(ctx, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if got := received.Load(); got != 0 {
		t.Errorf("expected 0 deliveries for non-matching subject, got %d", got)
	}
}

func TestMemoryBus_NonMatchingStream(t *testing.T) {
	b := NewMemoryBus()
	ctx := context.Background()

	var received atomic.Int32
	sub, err := b.Subscribe(ctx, "stream-a", ">", func(_ context.Context, env Envelope) error {
		received.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	env := Envelope{
		ID:      "550e8400-e29b-41d4-a716-446655440000",
		Stream:  "stream-b",
		Type:    "test.event",
		TS:      time.Now().UTC(),
		Subject: "test:123",
		Payload: json.RawMessage(`{}`),
		Version: 1,
	}

	if err := b.Publish(ctx, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if got := received.Load(); got != 0 {
		t.Errorf("expected 0 deliveries for non-matching stream, got %d", got)
	}
}

func TestMemoryBus_MultipleSubscribers(t *testing.T) {
	b := NewMemoryBus()
	ctx := context.Background()

	var count1, count2 atomic.Int32
	sub1, err := b.Subscribe(ctx, "s", ">", func(_ context.Context, env Envelope) error {
		count1.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe 1: %v", err)
	}
	defer sub1.Close()

	sub2, err := b.Subscribe(ctx, "s", ">", func(_ context.Context, env Envelope) error {
		count2.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe 2: %v", err)
	}
	defer sub2.Close()

	env := Envelope{
		ID:      "550e8400-e29b-41d4-a716-446655440000",
		Stream:  "s",
		Type:    "t",
		TS:      time.Now().UTC(),
		Subject: "x",
		Payload: json.RawMessage(`{}`),
		Version: 1,
	}

	if err := b.Publish(ctx, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if g1, g2 := count1.Load(), count2.Load(); g1 != 1 || g2 != 1 {
		t.Errorf("expected both handlers to fire (1,1), got (%d,%d)", g1, g2)
	}
}

func TestMemoryBus_SubscribeClose(t *testing.T) {
	b := NewMemoryBus()
	ctx := context.Background()

	var received atomic.Int32
	sub, err := b.Subscribe(ctx, "s", ">", func(_ context.Context, env Envelope) error {
		received.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	env := Envelope{
		ID:      "550e8400-e29b-41d4-a716-446655440000",
		Stream:  "s",
		Type:    "t",
		TS:      time.Now().UTC(),
		Subject: "x",
		Payload: json.RawMessage(`{}`),
		Version: 1,
	}

	if err := b.Publish(ctx, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if got := received.Load(); got != 0 {
		t.Errorf("expected 0 after unsubscribing, got %d", got)
	}
}

func TestMemoryBus_InvalidEnvelope(t *testing.T) {
	b := NewMemoryBus()
	ctx := context.Background()

	var received atomic.Int32
	sub, err := b.Subscribe(ctx, "s", ">", func(_ context.Context, env Envelope) error {
		received.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	invalid := Envelope{ID: "", Stream: "", Type: "", Subject: ""}
	err = b.Publish(ctx, invalid)
	if err == nil {
		t.Error("expected error for invalid envelope")
	}
	if got := received.Load(); got != 0 {
		t.Errorf("expected 0 deliveries for invalid envelope, got %d", got)
	}
}

func TestMemoryBus_SubscribeNilHandler(t *testing.T) {
	b := NewMemoryBus()
	ctx := context.Background()

	_, err := b.Subscribe(ctx, "s", ">", nil)
	if err == nil {
		t.Error("expected error for nil handler")
	}
}

func TestMemoryBus_SubscribeEmptyStream(t *testing.T) {
	b := NewMemoryBus()
	ctx := context.Background()

	_, err := b.Subscribe(ctx, "", ">", func(_ context.Context, env Envelope) error {
		return nil
	})
	if err == nil {
		t.Error("expected error for empty stream")
	}
}

func TestIdempotent(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	var count atomic.Int32
	h := Idempotent(store, func(_ context.Context, env Envelope) error {
		count.Add(1)
		return nil
	})

	env := Envelope{
		ID:      "550e8400-e29b-41d4-a716-446655440000",
		Stream:  "s",
		Type:    "t",
		TS:      time.Now().UTC(),
		Subject: "x",
		Payload: json.RawMessage(`{}`),
		Version: 1,
	}

	// first call
	if err := h(ctx, env); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 execution, got %d", got)
	}

	// duplicate delivery — should be skipped
	if err := h(ctx, env); err != nil {
		t.Fatalf("duplicate call: %v", err)
	}
	if got := count.Load(); got != 1 {
		t.Errorf("expected still 1 execution after dedupe, got %d", got)
	}
}

func TestMatchSubject_Exact(t *testing.T) {
	if !matchSubject("foo.bar", "foo.bar") {
		t.Error("expected exact match")
	}
	if matchSubject("foo.bar", "foo.baz") {
		t.Error("expected no match")
	}
}

func TestMatchSubject_Wildcard(t *testing.T) {
	if !matchSubject(">", "anything.here") {
		t.Error("expected '>' to match everything")
	}
	if !matchSubject("", "anything.here") {
		t.Error("expected empty to match everything")
	}
}

func TestMatchSubject_Prefix(t *testing.T) {
	if !matchSubject("foo.*", "foo.bar") {
		t.Error("expected foo.* to match foo.bar")
	}
	if !matchSubject("foo.*", "foo.bar.baz") {
		t.Error("expected foo.* to match foo.bar.baz (prefix match)")
	}
	if matchSubject("foo.*", "foobar.baz") {
		t.Error("expected foo.* not to match foobar.baz")
	}
	if matchSubject("foo.*", "other.bar") {
		t.Error("expected foo.* not to match other.bar")
	}
}
