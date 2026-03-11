package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
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
	permSvc     *service.PermissionPolicyService
	groupRepo   repository.GroupRepo
	userRepo    repository.UserRepo
}

// NewHub creates a new Hub.
func NewHub(
	ps pubsub.PubSub,
	locationSvc *service.LocationService,
	permSvc *service.PermissionPolicyService,
	groupRepo repository.GroupRepo,
	userRepo repository.UserRepo,
) *Hub {
	return &Hub{
		clients:     make(map[uuid.UUID]map[*Client]struct{}),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		ps:          ps,
		locationSvc: locationSvc,
		permSvc:     permSvc,
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
		"group:*:streams",
		"group:*:ptt",
		"user:*:location",
		"user:*:membership",
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
		"groups", client.membershipCount(),
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
		// Parse the broadcast payload — shared by both group and user channels.
		var broadcast service.LocationBroadcast
		if err := json.Unmarshal(msg.Payload, &broadcast); err != nil {
			slog.Warn("failed to unmarshal location broadcast", "error", err)
			return
		}

		// Build server → client envelope once; reused for all recipients.
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

		if strings.HasPrefix(channel, "group:") {
			// channel format: group:<uuid>:location
			groupID := extractID(channel)
			if groupID == uuid.Nil {
				return
			}

			slog.Debug("routeMessage: location broadcast parsed",
				"group_id", groupID,
				"sender_user_id", broadcast.UserID,
				"sender_device_id", broadcast.DeviceID,
				"lat", broadcast.Lat,
				"lng", broadcast.Lng,
				"total_connected_clients", h.totalClients(),
			)

			// Send to all group members with view_locations permission,
			// excluding the sending device (but including the sender's other devices).
			recipientCount := h.broadcastToGroupExcludeDevice(groupID, broadcast.DeviceID, model.ActionViewLocations, data)
			slog.Debug("routeMessage: location broadcast sent",
				"group_id", groupID,
				"recipients", recipientCount,
			)
		} else if strings.HasPrefix(channel, "user:") {
			// channel format: user:<uuid>:location
			// Self-broadcast: deliver to the user's other connected devices only.
			userID := extractID(channel)
			if userID == uuid.Nil {
				return
			}

			slog.Debug("routeMessage: self location broadcast",
				"user_id", userID,
				"sender_device_id", broadcast.DeviceID,
				"lat", broadcast.Lat,
				"lng", broadcast.Lng,
			)

			recipientCount := h.sendLocationToUser(userID, broadcast.DeviceID, data)
			slog.Debug("routeMessage: self location broadcast sent",
				"user_id", userID,
				"recipients", recipientCount,
			)
		}

	case strings.HasPrefix(channel, "user:") && strings.HasSuffix(channel, ":membership"):
		// channel format: user:<uuid>:membership
		// A user's group memberships changed — refresh all their connected clients.
		userID := extractID(channel)
		if userID == uuid.Nil {
			return
		}
		slog.Debug("routeMessage: membership change received", "user_id", userID)
		h.refreshClientMemberships(context.Background(), userID)

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

		recipientCount := h.broadcastToGroup(groupID, partial.SenderID, model.ActionReadMessages, data)
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

	case strings.HasSuffix(channel, ":streams"):
		// channel format: group:<uuid>:streams
		// Payload must include "event_type" to determine the WS message type.
		groupID := extractID(channel)
		if groupID == uuid.Nil {
			return
		}

		var partial struct {
			EventType string    `json:"event_type"`
			UserID    uuid.UUID `json:"user_id"`
		}
		if err := json.Unmarshal(msg.Payload, &partial); err != nil {
			slog.Warn("failed to unmarshal stream event", "error", err)
			return
		}

		wsType := partial.EventType
		if wsType == "" {
			slog.Warn("stream event missing event_type", "channel", channel)
			return
		}

		data, err := json.Marshal(Envelope{
			Type:    wsType,
			Payload: msg.Payload,
		})
		if err != nil {
			slog.Warn("failed to marshal stream envelope", "error", err)
			return
		}

		recipientCount := h.broadcastToGroup(groupID, uuid.Nil, model.ActionViewStream, data)
		slog.Debug("routeMessage: stream event broadcast",
			"group_id", groupID,
			"event_type", wsType,
			"recipients", recipientCount,
		)

	case strings.HasSuffix(channel, ":ptt"):
		// channel format: group:<uuid>:ptt
		// Payload must include "event_type" to determine the WS message type.
		groupID := extractID(channel)
		if groupID == uuid.Nil {
			return
		}

		var partial struct {
			EventType string    `json:"event_type"`
			UserID    uuid.UUID `json:"user_id"`
		}
		if err := json.Unmarshal(msg.Payload, &partial); err != nil {
			slog.Warn("failed to unmarshal ptt event", "error", err)
			return
		}

		wsType := partial.EventType
		if wsType == "" {
			slog.Warn("ptt event missing event_type", "channel", channel)
			return
		}

		data, err := json.Marshal(Envelope{
			Type:    wsType,
			Payload: msg.Payload,
		})
		if err != nil {
			slog.Warn("failed to marshal ptt envelope", "error", err)
			return
		}

		recipientCount := h.broadcastToGroup(groupID, uuid.Nil, model.ActionUsePTT, data)
		slog.Debug("routeMessage: ptt event broadcast",
			"group_id", groupID,
			"event_type", wsType,
			"recipients", recipientCount,
		)

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
			recipientCount := h.broadcastToGroup(targetID, partial.OwnerID, model.ActionReadMessages, data)
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
// the given group and have the required permission, optionally excluding a
// specific user (the sender).
// Returns the number of clients the message was sent to.
func (h *Hub) broadcastToGroup(groupID uuid.UUID, excludeUserID uuid.UUID, requiredAction string, data []byte) int {
	policy, _ := h.permSvc.GetPolicy(context.Background())

	sent := 0
	for userID, clients := range h.clients {
		if userID == excludeUserID {
			continue
		}
		for client := range clients {
			member := client.membership(groupID)
			if member == nil {
				continue
			}
			if policy != nil && !policy.CheckCommunication(requiredAction, member, client.isAdmin) {
				continue
			}
			select {
			case client.send <- data:
				sent++
			default:
				// Buffer full, skip
			}
		}
	}
	return sent
}

// broadcastToGroupExcludeDevice sends data to all connected clients who are
// members of the given group and have the required permission, excluding only
// the specific sending device. Used for location broadcasts so that the
// sender's other clients still receive the update.
// Returns the number of clients the message was sent to.
func (h *Hub) broadcastToGroupExcludeDevice(groupID uuid.UUID, excludeDeviceID uuid.UUID, requiredAction string, data []byte) int {
	policy, _ := h.permSvc.GetPolicy(context.Background())

	totalClients := h.totalClients()
	slog.Debug("broadcastToGroupExcludeDevice: starting fan-out",
		"group_id", groupID,
		"exclude_device_id", excludeDeviceID,
		"action", requiredAction,
		"total_connected_clients", totalClients,
	)

	sent := 0
	for _, clients := range h.clients {
		for client := range clients {
			if client.deviceID == excludeDeviceID {
				slog.Debug("broadcastToGroupExcludeDevice: skipping sender device",
					"client_user_id", client.userID,
					"client_device_id", client.deviceID,
				)
				continue
			}
			member := client.membership(groupID)
			if member == nil {
				slog.Debug("broadcastToGroupExcludeDevice: client not in group",
					"client_user_id", client.userID,
					"client_device_id", client.deviceID,
					"group_id", groupID,
				)
				continue
			}
			if policy != nil && !policy.CheckCommunication(requiredAction, member, client.isAdmin) {
				slog.Debug("broadcastToGroupExcludeDevice: permission denied",
					"client_user_id", client.userID,
					"client_device_id", client.deviceID,
					"group_id", groupID,
					"action", requiredAction,
					"can_read", member.CanRead,
					"can_write", member.CanWrite,
					"is_group_admin", member.IsGroupAdmin,
					"is_server_admin", client.isAdmin,
				)
				continue
			}
			select {
			case client.send <- data:
				sent++
				slog.Debug("broadcastToGroupExcludeDevice: delivered to client",
					"client_user_id", client.userID,
					"client_device_id", client.deviceID,
				)
			default:
				slog.Warn("broadcastToGroupExcludeDevice: client send buffer full, dropping location broadcast",
					"client_user_id", client.userID,
					"client_device_id", client.deviceID,
					"group_id", groupID,
				)
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

// sendLocationToUser delivers a location broadcast to all of the user's connected
// clients except the sending device. Used for the self-broadcast path so a user
// always sees their own other devices update in real-time.
func (h *Hub) sendLocationToUser(userID uuid.UUID, excludeDeviceID uuid.UUID, data []byte) int {
	clients, ok := h.clients[userID]
	if !ok {
		return 0
	}
	sent := 0
	for client := range clients {
		if client.deviceID == excludeDeviceID {
			continue
		}
		select {
		case client.send <- data:
			sent++
		default:
			slog.Warn("sendLocationToUser: client send buffer full, dropping self-broadcast",
				"user_id", userID,
				"device_id", client.deviceID,
			)
		}
	}
	return sent
}

// refreshClientMemberships reloads the group memberships for all connected
// clients of the given user. Called when the hub receives a membership-changed
// event from Redis. For any newly-joined group, a location snapshot is sent
// immediately so the client sees existing positions right away.
func (h *Hub) refreshClientMemberships(ctx context.Context, userID uuid.UUID) {
	clients, ok := h.clients[userID]
	if !ok {
		slog.Debug("refreshClientMemberships: no connected clients", "user_id", userID)
		return
	}

	memberships, err := h.groupRepo.ListMembershipsByUserID(ctx, userID)
	if err != nil {
		slog.Error("refreshClientMemberships: failed to reload memberships",
			"user_id", userID, "error", err)
		return
	}

	newMap := make(map[uuid.UUID]*model.GroupMember, len(memberships))
	for i := range memberships {
		newMap[memberships[i].GroupID] = &memberships[i]
	}

	slog.Debug("refreshClientMemberships: updating clients",
		"user_id", userID,
		"client_count", len(clients),
		"group_count", len(newMap),
	)

	policy, _ := h.permSvc.GetPolicy(ctx)

	for client := range clients {
		oldMap := client.setGroupMemberships(newMap)

		// For each newly joined group, send an immediate location snapshot.
		for groupID, member := range newMap {
			if _, wasPresent := oldMap[groupID]; wasPresent {
				continue
			}
			if policy != nil && !policy.CheckCommunication(model.ActionViewLocations, member, client.isAdmin) {
				continue
			}
			records, err := h.locationSvc.GetGroupSnapshot(ctx, groupID)
			if err != nil {
				slog.Error("refreshClientMemberships: failed to get snapshot",
					"group_id", groupID, "error", err)
				continue
			}
			locations := h.buildSnapshotLocations(records, client.deviceID, groupID)
			if len(locations) > 0 {
				client.sendJSON(TypeLocationSnapshot, LocationSnapshotPayload{
					GroupID:   groupID,
					Locations: locations,
				})
			}
		}

		// Re-send connected so the client knows its updated group list.
		h.SendConnected(ctx, client)
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

	// Filter groups to only those where the user has share_location permission
	policy, _ := h.permSvc.GetPolicy(ctx)
	allGroupIDs := c.groupIDs()
	filteredGroups := make([]uuid.UUID, 0, len(allGroupIDs))
	for _, gid := range allGroupIDs {
		member := c.membership(gid)
		if policy != nil && policy.CheckCommunication(model.ActionShareLocation, member, c.isAdmin) {
			filteredGroups = append(filteredGroups, gid)
		} else {
			slog.Debug("location_update: group filtered out by share_location permission",
				"user_id", c.userID,
				"device_id", c.deviceID,
				"group_id", gid,
			)
		}
	}

	slog.Debug("location_update: group filter result",
		"user_id", c.userID,
		"device_id", c.deviceID,
		"total_groups", len(allGroupIDs),
		"broadcast_groups", len(filteredGroups),
	)

	accepted, err := h.locationSvc.Update(
		ctx,
		c.userID, c.deviceID,
		c.username, displayName, c.deviceName,
		c.isPrimary,
		payload.Lat, payload.Lng,
		payload.Altitude, payload.Heading, payload.Speed, payload.Accuracy,
		filteredGroups,
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
		"groups", len(filteredGroups),
	)
}

// handlePTTFloorRequest publishes a PTT floor request to Redis so it reaches
// all API nodes. The actual floor arbitration will be handled by the PTT
// service in a later phase.
func (h *Hub) handlePTTFloorRequest(ctx context.Context, c *Client, payload *PTTFloorRequestPayload) {
	// Find the group that owns this channel. For now, use the channel_id as
	// the group ID — the PTT service will resolve channels in Phase 4.
	groupID := payload.ChannelID

	member := c.membership(groupID)
	if member == nil {
		c.sendError("not a member of the group")
		return
	}

	policy, _ := h.permSvc.GetPolicy(ctx)
	if policy != nil && !policy.CheckCommunication(model.ActionUsePTT, member, c.isAdmin) {
		c.sendError("permission denied: use_ptt")
		return
	}

	event, err := json.Marshal(map[string]any{
		"event_type": TypePTTFloorRequest,
		"channel_id": payload.ChannelID,
		"user_id":    c.userID,
		"device_id":  c.deviceID,
	})
	if err != nil {
		slog.Error("failed to marshal ptt floor request", "error", err)
		return
	}

	channel := "group:" + groupID.String() + ":ptt"
	if err := h.ps.Publish(ctx, channel, event); err != nil {
		slog.Error("failed to publish ptt floor request", "error", err, "channel", channel)
		c.sendError("failed to relay ptt floor request")
	}
}

// handlePTTFloorRelease publishes a PTT floor release to Redis so it reaches
// all API nodes.
func (h *Hub) handlePTTFloorRelease(ctx context.Context, c *Client, payload *PTTFloorRequestPayload) {
	groupID := payload.ChannelID

	member := c.membership(groupID)
	if member == nil {
		c.sendError("not a member of the group")
		return
	}

	event, err := json.Marshal(map[string]any{
		"event_type": TypePTTFloorRelease,
		"channel_id": payload.ChannelID,
		"user_id":    c.userID,
		"device_id":  c.deviceID,
	})
	if err != nil {
		slog.Error("failed to marshal ptt floor release", "error", err)
		return
	}

	channel := "group:" + groupID.String() + ":ptt"
	if err := h.ps.Publish(ctx, channel, event); err != nil {
		slog.Error("failed to publish ptt floor release", "error", err, "channel", channel)
		c.sendError("failed to relay ptt floor release")
	}
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

// buildSnapshotLocations converts location records into LocationBroadcastPayload
// entries, skipping the given excludeDeviceID and tagging each with groupID.
func (h *Hub) buildSnapshotLocations(records []repository.LocationRecord, excludeDeviceID uuid.UUID, groupID uuid.UUID) []LocationBroadcastPayload {
	locations := make([]LocationBroadcastPayload, 0, len(records))
	for _, rec := range records {
		if rec.DeviceID == excludeDeviceID {
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
	return locations
}

// SendSnapshot sends the initial location snapshot for all of a client's groups
// where the client has view_locations permission, plus a self-snapshot of the
// user's own other devices.
func (h *Hub) SendSnapshot(ctx context.Context, client *Client) {
	policy, _ := h.permSvc.GetPolicy(ctx)

	// Group-based snapshot: one message per group the client belongs to.
	client.mu.RLock()
	memberships := make(map[uuid.UUID]*model.GroupMember, len(client.groupMemberships))
	for k, v := range client.groupMemberships {
		memberships[k] = v
	}
	client.mu.RUnlock()

	for groupID, member := range memberships {
		if policy != nil && !policy.CheckCommunication(model.ActionViewLocations, member, client.isAdmin) {
			continue
		}
		records, err := h.locationSvc.GetGroupSnapshot(ctx, groupID)
		if err != nil {
			slog.Error("failed to get group snapshot", "group_id", groupID, "error", err)
			continue
		}
		locations := h.buildSnapshotLocations(records, client.deviceID, groupID)
		if len(locations) > 0 {
			client.sendJSON(TypeLocationSnapshot, LocationSnapshotPayload{
				GroupID:   groupID,
				Locations: locations,
			})
		}
	}

	// Self-snapshot: the user's own other devices, regardless of group membership.
	// This ensures a user always sees their own devices update in real-time even
	// when they share no groups with them (e.g. freshly-created account).
	ownRecords, err := h.locationSvc.GetUserSnapshot(ctx, client.userID)
	if err != nil {
		slog.Error("failed to get user self-snapshot", "user_id", client.userID, "error", err)
	} else {
		locations := h.buildSnapshotLocations(ownRecords, client.deviceID, uuid.Nil)
		if len(locations) > 0 {
			client.sendJSON(TypeLocationSnapshot, LocationSnapshotPayload{
				GroupID:   uuid.Nil,
				Locations: locations,
			})
		}
	}
}

// SendConnected sends the "connected" acknowledgement with the current group list.
// Safe to call after a membership refresh as well as on initial connect.
func (h *Hub) SendConnected(ctx context.Context, client *Client) {
	client.mu.RLock()
	groupIDs := make([]uuid.UUID, 0, len(client.groupMemberships))
	for gid := range client.groupMemberships {
		groupIDs = append(groupIDs, gid)
	}
	client.mu.RUnlock()

	groups := make([]ConnectedGroup, 0, len(groupIDs))
	for _, gid := range groupIDs {
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
