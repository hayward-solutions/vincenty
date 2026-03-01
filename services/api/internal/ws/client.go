package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	"nhooyr.io/websocket"
)

const (
	// Maximum message size from client (64 KB).
	maxMessageSize = 64 * 1024

	// Time to wait for a pong before considering the connection dead.
	pongWait = 60 * time.Second

	// Send pings at this interval (must be < pongWait).
	pingInterval = 30 * time.Second

	// Write deadline for outgoing messages.
	writeWait = 10 * time.Second

	// Size of the outbound message buffer per client.
	sendBufferSize = 256
)

// Client represents a single WebSocket connection.
type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	userID     uuid.UUID
	deviceID   uuid.UUID
	deviceName string
	isPrimary  bool
	username   string
	isAdmin    bool

	// groupMemberships is protected by mu because it is read by the client's
	// readPump goroutine (handleLocationUpdate) and written by the hub's event
	// loop goroutine (refreshClientMemberships).
	mu               sync.RWMutex
	groupMemberships map[uuid.UUID]*model.GroupMember

	send chan []byte
}

// NewClient creates a new Client.
func NewClient(hub *Hub, conn *websocket.Conn, userID, deviceID uuid.UUID, deviceName string, isPrimary bool, username string, isAdmin bool, memberships map[uuid.UUID]*model.GroupMember) *Client {
	return &Client{
		hub:              hub,
		conn:             conn,
		userID:           userID,
		deviceID:         deviceID,
		deviceName:       deviceName,
		isPrimary:        isPrimary,
		username:         username,
		isAdmin:          isAdmin,
		groupMemberships: memberships,
		send:             make(chan []byte, sendBufferSize),
	}
}

// groupIDs returns a snapshot of the current group UUIDs.
// Safe to call from any goroutine.
func (c *Client) groupIDs() []uuid.UUID {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids := make([]uuid.UUID, 0, len(c.groupMemberships))
	for id := range c.groupMemberships {
		ids = append(ids, id)
	}
	return ids
}

// membership returns the GroupMember for the given group, or nil if not a member.
// Safe to call from any goroutine.
func (c *Client) membership(groupID uuid.UUID) *model.GroupMember {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.groupMemberships[groupID]
}

// setGroupMemberships atomically replaces the membership map and returns the
// previous map so callers can diff old vs new.
// Must only be called from the hub's event loop goroutine.
func (c *Client) setGroupMemberships(m map[uuid.UUID]*model.GroupMember) map[uuid.UUID]*model.GroupMember {
	c.mu.Lock()
	defer c.mu.Unlock()
	old := c.groupMemberships
	c.groupMemberships = m
	return old
}

// membershipCount returns the number of groups the client belongs to.
// Safe to call from any goroutine.
func (c *Client) membershipCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.groupMemberships)
}

// Run starts the client's read and write pumps. It blocks until the
// connection is closed.
func (c *Client) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start write pump in background
	go c.writePump(ctx)

	// Read pump runs in foreground; when it returns, the client is done
	c.readPump(ctx)
}

// readPump reads messages from the WebSocket and dispatches them.
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	c.conn.SetReadLimit(maxMessageSize)

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			// Don't log normal closures
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				slog.Debug("client disconnected", "user_id", c.userID)
			} else {
				slog.Warn("ws read error", "user_id", c.userID, "error", err)
			}
			return
		}

		// Parse the envelope
		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			c.sendError("invalid message format")
			continue
		}

		c.handleMessage(ctx, &env)
	}
}

// writePump drains the send channel and writes messages to the WebSocket.
// It also sends periodic pings to keep the connection alive.
//
// The pump exits only when the send channel is closed (by the hub on
// unregister or shutdown). This guarantees a WebSocket close frame is
// always sent, avoiding the race condition that existed when ctx.Done()
// was also a select case.
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// Hub closed the channel — send a close frame and exit.
				c.conn.Close(websocket.StatusGoingAway, "server shutting down")
				return
			}

			writeCtx, cancel := context.WithTimeout(ctx, writeWait)
			err := c.conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				slog.Warn("ws write error", "user_id", c.userID, "error", err)
				return
			}

		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeWait)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				slog.Debug("ping failed, closing client", "user_id", c.userID, "error", err)
				return
			}
		}
	}
}

// handleMessage routes an incoming message to the appropriate handler.
func (c *Client) handleMessage(ctx context.Context, env *Envelope) {
	switch env.Type {
	case TypeLocationUpdate:
		var payload LocationUpdatePayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			c.sendError("invalid location_update payload")
			return
		}
		c.hub.handleLocationUpdate(ctx, c, &payload)

	default:
		c.sendError("unknown message type: " + env.Type)
	}
}

// sendError sends an error message to the client.
func (c *Client) sendError(message string) {
	data, err := NewEnvelope(TypeError, ErrorPayload{Message: message})
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		// Buffer full, drop the message
	}
}

// sendJSON sends a typed message to the client.
func (c *Client) sendJSON(msgType string, payload any) {
	data, err := NewEnvelope(msgType, payload)
	if err != nil {
		slog.Error("failed to marshal ws message", "type", msgType, "error", err)
		return
	}
	select {
	case c.send <- data:
	default:
		slog.Warn("client send buffer full, dropping message", "user_id", c.userID, "type", msgType)
	}
}
