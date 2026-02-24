package pubsub

import (
	"context"
	"sync"
)

// PublishedMessage records a single Publish call for test assertions.
type PublishedMessage struct {
	Channel string
	Payload []byte
}

// MockPubSub is an in‑memory PubSub implementation for testing.
// It records all published messages and allows tests to inspect them.
type MockPubSub struct {
	mu        sync.Mutex
	published []PublishedMessage
	subCh     chan Message
	err       error // if non-nil, Publish/Subscribe return this error
}

// NewMockPubSub creates a ready‑to‑use MockPubSub.
func NewMockPubSub() *MockPubSub {
	return &MockPubSub{
		subCh: make(chan Message, 256),
	}
}

// SetError makes all subsequent Publish/Subscribe calls return err.
func (m *MockPubSub) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Publish records the message and optionally fans it out to subscribers.
func (m *MockPubSub) Publish(_ context.Context, channel string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	cp := make([]byte, len(payload))
	copy(cp, payload)
	m.published = append(m.published, PublishedMessage{Channel: channel, Payload: cp})
	// Best‐effort fan‐out so hub tests can observe broadcasts.
	select {
	case m.subCh <- Message{Channel: channel, Payload: cp}:
	default:
	}
	return nil
}

// Subscribe returns the internal message channel.
func (m *MockPubSub) Subscribe(_ context.Context, _ ...string) (<-chan Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	return m.subCh, nil
}

// Close is a no‐op for the mock.
func (m *MockPubSub) Close() error { return nil }

// Published returns a snapshot of all published messages.
func (m *MockPubSub) Published() []PublishedMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]PublishedMessage, len(m.published))
	copy(out, m.published)
	return out
}

// Reset clears all recorded messages.
func (m *MockPubSub) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = nil
}
