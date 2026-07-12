package aisdlc

import (
	"context"
	"errors"
	"sync"
)

// Bus publishes and subscribes to event envelopes.
//
// The M0 skeleton ships an in-process implementation for tests; the NATS
// JetStream implementation (deps: nats.go, go.opentelemetry.io/otel) arrives
// in M1. The interface is stable.
type Bus interface {
	Publish(ctx context.Context, env Envelope) error
	Subscribe(ctx context.Context, stream, subject string, handler Handler) (Subscription, error)
}

// Handler processes one envelope. A non-nil error NAKs the message so JetStream
// redelivers it; the consumer must therefore be idempotent — wrap with
// Idempotent.
type Handler func(ctx context.Context, env Envelope) error

// Subscription is an active subscription; Close stops delivery.
type Subscription interface {
	Close() error
}

// DedupeStore records envelope IDs that have already been processed. Used by
// Idempotent to make consumers exactly-once-effective under redelivery.
type DedupeStore interface {
	// Seen reports whether id was already recorded. If not, it records it and
	// returns (false, nil).
	Seen(id string) (bool, error)
}

// Idempotent wraps a Handler so duplicate deliveries (same envelope.ID) are
// skipped. Every consumer MUST be idempotent (ARCHITECTURE.md §6).
func Idempotent(store DedupeStore, h Handler) Handler {
	return func(ctx context.Context, env Envelope) error {
		seen, err := store.Seen(env.ID)
		if err != nil {
			return err
		}
		if seen {
			return nil
		}
		return h(ctx, env)
	}
}

// MemoryStore is an in-memory DedupeStore for tests/dev only (not durable).
type MemoryStore struct {
	mu sync.Mutex
	m  map[string]struct{}
}

// NewMemoryStore returns an empty in-memory dedupe store.
func NewMemoryStore() *MemoryStore { return &MemoryStore{m: make(map[string]struct{})} }

// Seen implements DedupeStore.
func (s *MemoryStore) Seen(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[id]; ok {
		return true, nil
	}
	s.m[id] = struct{}{}
	return false, nil
}

// MemoryBus is an in-process Bus for unit tests (M0). Not for production.
type MemoryBus struct {
	mu   sync.Mutex
	subs []memorySub
}

type memorySub struct {
	stream, pattern string
	h               Handler
}

// NewMemoryBus returns an empty in-process bus.
func NewMemoryBus() *MemoryBus { return &MemoryBus{} }

// Publish validates the envelope and dispatches it to matching subscribers.
func (b *MemoryBus) Publish(ctx context.Context, env Envelope) error {
	if err := env.Validate(); err != nil {
		return err
	}
	b.mu.Lock()
	subs := append([]memorySub(nil), b.subs...)
	b.mu.Unlock()
	for _, s := range subs {
		if s.stream == env.Stream && matchSubject(s.pattern, env.Subject) {
			if err := s.h(ctx, env); err != nil {
				return err
			}
		}
	}
	return nil
}

// Subscribe registers a handler for a stream + subject pattern.
func (b *MemoryBus) Subscribe(_ context.Context, stream, subject string, h Handler) (Subscription, error) {
	if stream == "" || h == nil {
		return nil, errors.New("aisdlc: stream and handler are required")
	}
	b.mu.Lock()
	b.subs = append(b.subs, memorySub{stream, subject, h})
	b.mu.Unlock()
	return noopSub{}, nil
}

type noopSub struct{}

func (noopSub) Close() error { return nil }

// matchSubject supports exact match, empty/">" (all), and a trailing ".*"
// wildcard. A minimal subset of NATS tokens for the in-memory bus.
func matchSubject(pattern, subject string) bool {
	if pattern == "" || pattern == ">" {
		return true
	}
	if pattern == subject {
		return true
	}
	if n := len(pattern); n > 2 && pattern[n-2:n] == ".*" {
		return len(subject) > n-2 && subject[:n-2] == pattern[:n-2]
	}
	return false
}
