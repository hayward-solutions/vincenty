package pubsub

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// RedisPubSub implements PubSub using Redis Pub/Sub.
type RedisPubSub struct {
	client *redis.Client
	sub    *redis.PubSub
}

// NewRedisPubSub creates a new Redis-backed PubSub.
func NewRedisPubSub(client *redis.Client) *RedisPubSub {
	return &RedisPubSub{client: client}
}

// Publish sends a message to the given Redis channel.
func (r *RedisPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	return r.client.Publish(ctx, channel, payload).Err()
}

// Subscribe uses PSUBSCRIBE to listen on glob patterns and returns a channel
// of incoming messages. The returned channel is closed when ctx is cancelled
// or Close is called.
func (r *RedisPubSub) Subscribe(ctx context.Context, patterns ...string) (<-chan Message, error) {
	r.sub = r.client.PSubscribe(ctx, patterns...)

	// Wait for confirmation of subscription
	if _, err := r.sub.Receive(ctx); err != nil {
		r.sub.Close()
		return nil, err
	}

	slog.Info("redis pubsub subscribed", "patterns", patterns)

	out := make(chan Message, 256)

	go func() {
		defer close(out)
		ch := r.sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				select {
				case out <- Message{
					Channel: msg.Channel,
					Payload: []byte(msg.Payload),
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

// Close shuts down the Redis subscription.
func (r *RedisPubSub) Close() error {
	if r.sub != nil {
		return r.sub.Close()
	}
	return nil
}
