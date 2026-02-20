// Package pubsub defines a pluggable publish/subscribe interface.
// The initial implementation uses Redis; future implementations may use
// Kafka or Apache Ignite.
package pubsub

import "context"

// Message represents a message received from a pub/sub channel.
type Message struct {
	// Channel is the channel/pattern the message was published to.
	Channel string
	// Payload is the raw message bytes.
	Payload []byte
}

// PubSub is the interface for publish/subscribe messaging.
type PubSub interface {
	// Publish sends a message to the given channel.
	Publish(ctx context.Context, channel string, payload []byte) error

	// Subscribe listens on the given glob patterns and returns a channel
	// that receives messages. The returned channel is closed when the
	// context is cancelled or Close is called.
	Subscribe(ctx context.Context, patterns ...string) (<-chan Message, error)

	// Close shuts down the pub/sub connection and releases resources.
	Close() error
}
