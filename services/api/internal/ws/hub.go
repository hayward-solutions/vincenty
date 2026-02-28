package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/pubsub"
	"github.com/sitaware/api/internal/repository"
	"github.com/sitaware/api/internal/service"
)

// Hub manages all active WebSocket clients and routes messages between them
// via Redis pub/sub.
type Hub struct {
	// Connected clients, keyed by user ID. One user may have multiple
	// clients (e.g., phone + browser).
	clients map[uuid.UUID]map[*Client]struct{}

	register   chan *Client
	unregister chan *Client

	ps          pubsub.PubSub
	locationSvc *service.LocationService
	groupRepo   repository.GroupRepo
	userRepo    repository.UserRepo
}

// NewHub creates a new Hub.
func NewHub(
	ps pubsub.PubSub,
	locationSvc *service.LocationService,
	groupRepo repository.GroupRepo,
	userRepo repository.UserRepo,
) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		ps:         ps,
		locationSvc: locationSvc,
		groupRepo:   groupRepo,
		userRepo:    userRepo,
	}
}

// Run starts the hub event loop. It blocks until ctx is cancelled.
func (h *Hub) Run(ctx context.Context) {
	slog.Info("websocket hub started")

	// Subscribe to all relevant Redis channels using pattern matching
	messages, err := h.ps.Subscribe(ctx,
		"group:*:location",
		"group:*:messages",
		"group:*:drawings",
		"user:*:direct",
		"user:*:drawings",
	)
	if err != nil {
		slog.Error("failed to subscribe to pubsub", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("websocket hub stopping, draining clients", "total", h.totalClients())
			// Close all client send channels. The writePump will detect
			// the closed channel, send a WebSocket close frame, and exit.
			for _, clients := range h.clients {
				for client := range clients {
					close(client.send)
				}
			}
			// Give write pumps a moment to flush close frames.
			time.Sleep(2 * time.Second)
			slog.Info("websocket hub stopped")
			return

		case client := <-h.register:
			h.addClient(client)

		case client := <-h.unregister:
			h.removeClient(client)

		case msg, ok := <-messages:
			if !ok {
				slog.Warn("pubsub channel closed")
				return
			}
			h.routeMessage(msg)
		}
	}
}

// addClient registers a new client with the hub.
func (h *Hub) addClient(client *Client) {
	if _, ok := h.clients[client.userID]; !ok {
		h.clients[client.userID] = make(map[*Client]struct{})
	}
	h.clients[client.userID][client] = struct{}{}

	slog.Info("client connected",
		"user_id", client.userID,
		"device_id", client.deviceID,
		"groups", len(client.groups),
		"total_clients", h.totalClients(),
	)
}

// removeClient unregisters a client from the hub.
func (h *Hub) removeClient(client *Client) {
	if clients, ok := h.clients[client.userID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.send)
		}
		if len(clients) == 0 {
			delete(h.clients, client.userID)
		}
	}

	slog.Info("client disconnected",
		"user_id", client.userID,
		"device_id", client.deviceID,
		"total_clients", h.totalClients(),
	)
}

// routeMessage takes a Redis pub/sub message and delivers it to the
// appropriate connected clients.
func (h *Hub) routeMessage(msg pubsub.Message) {
	channel := msg.Channel
	slog.Debug("routeMessage: received from Redis pub/sub",
		"channel", channel,
		"payload_bytes", len(msg.Payload),
	)

	switch {
	case strings.HasSuffix(channel, ":location"):
		// channel format: group:<uuid>:location
		groupID := extractID(channel)
		if groupID == uuid.Nil {
			return
		}

		// Parse the broadcast to get the sender's user ID
		var broadcast service.LocationBroadcast
		if err := json.Unmarshal(msg.Payload, &broadcast); err != nil {
			slog.Warn("failed to unmarshal location broadcast", "error", err)
			return
		}

		slog.Debug("routeMessage: location broadcast parsed",
			"group_id", groupID,
			"sender_user_id", broadcast.UserID,
			"lat", broadcast.Lat,
			"lng", broadcast.Lng,
		)

		// Build server → client envelope
		outPayload := LocationBroadcastPayload{
			UserID:      broadcast.UserID,
			Username:    broadcast.Username,
			DisplayName: broadcast.DisplayName,
			DeviceID:    broadcast.DeviceID,
			DeviceName:  broadcast.DeviceName,
			IsPrimary:   broadcast.IsPrimary,
			GroupID:     broadcast.GroupID,
			Lat:         broadcast.Lat,
			Lng:         broadcast.Lng,
			Altitude:    broadcast.Altitude,
			Heading:     broadcast.Heading,
			Speed:       broadcast.Speed,
			Timestamp:   broadcast.Timestamp,
		}

		data, err := NewEnvelope(TypeLocationBroadcast, outPayload)
		if err != nil {
			return
		}

		// Send to all clients who are members of this group,
		// except the sender themselves.
		recipientCount := h.broadcastToGroup(groupID, broadcast.UserID, data)
		slog.Debug("routeMessage: location broadcast sent",
			"group_id", groupID,
			"recipients", recipientCount,
		)

	case strings.HasSuffix(channel, ":messages"):
		// channel format: group:<uuid>:messages
		// Payload is a MessageResponse JSON published by MessageService.
		groupID := extractID(channel)
		if groupID == uuid.Nil {
			return
		}

		// Extract sender_id from the payload so we can exclude them
		var partial struct {
			SenderID uuid.UUID `json:"sender_id"`
		}
		if err := json.Unmarshal(msg.Payload, &partial); err != nil {
			slog.Warn("failed to unmarshal message broadcast sender", "error", err)
			return
		}

		// Wrap the raw MessageResponse in a WS envelope
		data, err := json.Marshal(Envelope{
			Type:    TypeMessageNew,
			Payload: msg.Payload,
		})
		if err != nil {
			slog.Warn("failed to marshal message envelope", "error", err)
			return
		}

		recipientCount := h.broadcastToGroup(groupID, partial.SenderID, data)
		slog.Debug("routeMessage: group message broadcast",
			"group_id", groupID,
			"sender_id", partial.SenderID,
			"recipients", recipientCount,
		)

	case strings.Contains(channel, "user:") && strings.HasSuffix(channel, ":direct"):
		// channel format: user:<uuid>:direct
		// Payload is a MessageResponse JSON published by MessageService.
		userID := extractID(channel)
		if userID == uuid.Nil {
			return
		}

		// Wrap in envelope
		data, err := json.Marshal(Envelope{
			Type:    TypeMessageNew,
			Payload: msg.Payload,
		})
		if err != nil {
			slog.Warn("failed to marshal direct message envelope", "error", err)
			return
		}

		slog.Debug("routeMessage: direct message",
			"target_user_id", userID,
		)
		h.sendToUser(userID, data)

	case strings.HasSuffix(channel, ":drawings"):
		// channel format: group:<uuid>:drawings OR user:<uuid>:drawings
		// Payload is a DrawingResponse JSON published by DrawingService.
		targetID := extractID(channel)
		if targetID == uuid.Nil {
			return
		}

		// Extract owner_id from the payload so we can exclude them
		var partial struct {
			OwnerID uuid.UUID `json:"owner_id"`
		}
		if err := json.Unmarshal(msg.Payload, &partial); err != nil {
			slog.Warn("failed to unmarshal drawing broadcast owner", "error", err)
			return
		}

		// Wrap the DrawingResponse in a WS envelope
		data, err := json.Marshal(Envelope{
			Type:    TypeDrawingUpdated,
			Payload: msg.Payload,
		})
		if err != nil {
			slog.Warn("failed to marshal drawing update envelope", "error", err)
			return
		}

		if strings.HasPrefix(channel, "group:") {
			recipientCount := h.broadcastToGroup(targetID, partial.OwnerID, data)
			slog.Debug("routeMessage: group drawing update",
				"group_id", targetID,
				"owner_id", partial.OwnerID,
				"recipients", recipientCount,
			)
		} else {
			// user:<uuid>:drawings — send to the specific user
			slog.Debug("routeMessage: direct drawing update",
				"target_user_id", targetID,
			)
			h.sendToUser(targetID, data)
		}
	}
}

// broadcastToGroup sends data to all connected clients who are members of
// the given group, optionally excluding a specific user (the sender).
// Returns the number of clients the message was sent to.
func (h *Hub) broadcastToGroup(groupID uuid.UUID, excludeUserID uuid.UUID, data []byte) int {
	sent := 0
	for userID, clients := range h.clients {
		if userID == excludeUserID {
			continue
		}
		for client := range clients {
			if isMember(client.groups, groupID) {
				select {
				case client.send <- data:
					sent++
				default:
					// Buffer full, skip
				}
			}
		}
	}
	return sent
}

// sendToUser sends data to all connected clients of a specific user.
func (h *Hub) sendToUser(userID uuid.UUID, data []byte) {
	if clients, ok := h.clients[userID]; ok {
		for client := range clients {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

// handleLocationUpdate processes an incoming location_update from a client.
func (h *Hub) handleLocationUpdate(ctx context.Context, c *Client, payload *LocationUpdatePayload) {
	slog.Debug("location_update received from client",
		"user_id", c.userID,
		"device_id", c.deviceID,
		"lat", payload.Lat,
		"lng", payload.Lng,
	)

	// Validate device ID matches the client's registered device
	deviceID, err := uuid.Parse(payload.DeviceID)
	if err != nil || deviceID != c.deviceID {
		slog.Debug("location_update rejected: device_id mismatch",
			"payload_device_id", payload.DeviceID,
			"client_device_id", c.deviceID,
		)
		c.sendError("device_id does not match authenticated device")
		return
	}

	// Get user display name for broadcast
	displayName := ""
	user, err := h.userRepo.GetByID(ctx, c.userID)
	if err == nil && user.DisplayName != nil {
		displayName = *user.DisplayName
	}

	accepted, err := h.locationSvc.Update(
		ctx,
		c.userID, c.deviceID,
		c.username, displayName, c.deviceName,
		c.isPrimary,
		payload.Lat, payload.Lng,
		payload.Altitude, payload.Heading, payload.Speed, payload.Accuracy,
		c.groups,
	)
	if err != nil {
		slog.Debug("location_update processing failed",
			"user_id", c.userID,
			"error", err,
		)
		c.sendError("failed to process location update")
		return
	}

	if !accepted {
		slog.Debug("location_update throttled",
			"user_id", c.userID,
			"device_id", c.deviceID,
		)
		return
	}

	slog.Debug("location_update accepted",
		"user_id", c.userID,
		"device_id", c.deviceID,
		"groups", len(c.groups),
	)
}

// totalClients returns the total number of connected clients.
func (h *Hub) totalClients() int {
	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}

// extractID extracts a UUID from a channel name formatted as "prefix:<uuid>:suffix".
func extractID(channel string) uuid.UUID {
	parts := strings.Split(channel, ":")
	if len(parts) < 3 {
		return uuid.Nil
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil
	}
	return id
}

// isMember checks if a UUID is in a slice.
func isMember(groups []uuid.UUID, id uuid.UUID) bool {
	for _, g := range groups {
		if g == id {
			return true
		}
	}
	return false
}

// SendSnapshot sends the initial location snapshot for all of a client's groups.
func (h *Hub) SendSnapshot(ctx context.Context, client *Client) {
	for _, groupID := range client.groups {
		records, err := h.locationSvc.GetGroupSnapshot(ctx, groupID)
		if err != nil {
			slog.Error("failed to get group snapshot", "group_id", groupID, "error", err)
			continue
		}

		locations := make([]LocationBroadcastPayload, 0, len(records))
		for _, rec := range records {
			// Don't include the client's own device in the snapshot
			if rec.DeviceID == client.deviceID {
				continue
			}
			dn := ""
			if rec.DisplayName != nil {
				dn = *rec.DisplayName
			}
			devName := ""
			if rec.DeviceName != nil {
				devName = *rec.DeviceName
			}
			locations = append(locations, LocationBroadcastPayload{
				UserID:      rec.UserID,
				Username:    rec.Username,
				DisplayName: dn,
				DeviceID:    rec.DeviceID,
				DeviceName:  devName,
				IsPrimary:   rec.IsPrimary,
				GroupID:     groupID,
				Lat:         rec.Lat,
				Lng:         rec.Lng,
				Altitude:    rec.Altitude,
				Heading:     rec.Heading,
				Speed:       rec.Speed,
				Timestamp:   rec.RecordedAt,
			})
		}

		if len(locations) > 0 {
			client.sendJSON(TypeLocationSnapshot, LocationSnapshotPayload{
				GroupID:   groupID,
				Locations: locations,
			})
		}
	}
}

// SendConnected sends the initial "connected" acknowledgement.
func (h *Hub) SendConnected(ctx context.Context, client *Client) {
	groups := make([]ConnectedGroup, 0, len(client.groups))
	for _, gid := range client.groups {
		group, err := h.groupRepo.GetByID(ctx, gid)
		if err != nil {
			continue
		}
		groups = append(groups, ConnectedGroup{
			ID:   group.ID,
			Name: group.Name,
		})
	}

	client.sendJSON(TypeConnected, ConnectedPayload{
		UserID: client.userID,
		Groups: groups,
	})
}
