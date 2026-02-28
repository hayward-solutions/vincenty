package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/pubsub"
	"github.com/sitaware/api/internal/repository"
	"github.com/sitaware/api/internal/repository/mock"
	"github.com/sitaware/api/internal/service"
)

// ---------------------------------------------------------------------------
// Pure function tests
// ---------------------------------------------------------------------------

func TestExtractID(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name    string
		channel string
		want    uuid.UUID
	}{
		{"group location", "group:" + id.String() + ":location", id},
		{"group messages", "group:" + id.String() + ":messages", id},
		{"user direct", "user:" + id.String() + ":direct", id},
		{"too few parts", "group:" + id.String(), uuid.Nil},
		{"invalid uuid", "group:not-a-uuid:location", uuid.Nil},
		{"empty", "", uuid.Nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractID(tt.channel)
			if got != tt.want {
				t.Errorf("extractID(%q) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Hub client management
// ---------------------------------------------------------------------------

func newTestHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]struct{}),
		register:   make(chan *Client, 10),
		unregister: make(chan *Client, 10),
	}
}

// newTestPermSvc creates a PermissionPolicyService backed by a mock that
// returns no stored policy (so DefaultPermissionPolicy is used).
func newTestPermSvc() *service.PermissionPolicyService {
	settingsRepo := &mock.ServerSettingsRepo{
		GetFn: func(ctx context.Context, key string) (*model.ServerSetting, error) {
			return nil, model.ErrNotFound("setting")
		},
	}
	return service.NewPermissionPolicyService(settingsRepo)
}

// makeMemberships builds a groupMemberships map with full permissions for the
// given group IDs.
func makeMemberships(groupIDs ...uuid.UUID) map[uuid.UUID]*model.GroupMember {
	m := make(map[uuid.UUID]*model.GroupMember, len(groupIDs))
	for _, id := range groupIDs {
		m[id] = &model.GroupMember{
			GroupID:      id,
			CanRead:      true,
			CanWrite:     true,
			IsGroupAdmin: false,
		}
	}
	return m
}

func newTestClient(hub *Hub, userID, deviceID uuid.UUID, groupIDs []uuid.UUID) *Client {
	return &Client{
		hub:              hub,
		userID:           userID,
		deviceID:         deviceID,
		groupMemberships: makeMemberships(groupIDs...),
		send:             make(chan []byte, sendBufferSize),
		username:         "testuser",
	}
}

func TestHub_AddRemoveClient(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	deviceID := uuid.New()
	client := newTestClient(hub, userID, deviceID, nil)

	hub.addClient(client)

	if hub.totalClients() != 1 {
		t.Errorf("expected 1 client, got %d", hub.totalClients())
	}
	if _, ok := hub.clients[userID]; !ok {
		t.Error("expected user to be in clients map")
	}

	hub.removeClient(client)

	if hub.totalClients() != 0 {
		t.Errorf("expected 0 clients, got %d", hub.totalClients())
	}
	if _, ok := hub.clients[userID]; ok {
		t.Error("expected user to be removed from clients map")
	}
}

func TestHub_MultipleClientsPerUser(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	c1 := newTestClient(hub, userID, uuid.New(), nil)
	c2 := newTestClient(hub, userID, uuid.New(), nil)

	hub.addClient(c1)
	hub.addClient(c2)

	if hub.totalClients() != 2 {
		t.Errorf("expected 2 clients, got %d", hub.totalClients())
	}

	// Remove first client — user should still be in map
	hub.removeClient(c1)
	if hub.totalClients() != 1 {
		t.Errorf("expected 1 client, got %d", hub.totalClients())
	}
	if _, ok := hub.clients[userID]; !ok {
		t.Error("expected user still in clients map with remaining client")
	}

	// Remove second — user should be fully removed
	hub.removeClient(c2)
	if hub.totalClients() != 0 {
		t.Errorf("expected 0 clients, got %d", hub.totalClients())
	}
}

func TestHub_RemoveClientIdempotent(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	client := newTestClient(hub, userID, uuid.New(), nil)

	hub.addClient(client)
	hub.removeClient(client) // closes send channel
	hub.removeClient(client) // should not panic
}

// ---------------------------------------------------------------------------
// broadcastToGroup
// ---------------------------------------------------------------------------

func TestHub_BroadcastToGroup(t *testing.T) {
	hub := newTestHub()
	hub.permSvc = newTestPermSvc()
	groupID := uuid.New()
	senderID := uuid.New()
	recipientID := uuid.New()
	outsiderID := uuid.New()

	sender := newTestClient(hub, senderID, uuid.New(), []uuid.UUID{groupID})
	recipient := newTestClient(hub, recipientID, uuid.New(), []uuid.UUID{groupID})
	outsider := newTestClient(hub, outsiderID, uuid.New(), []uuid.UUID{uuid.New()})

	hub.addClient(sender)
	hub.addClient(recipient)
	hub.addClient(outsider)

	data := []byte(`{"test":"broadcast"}`)
	sent := hub.broadcastToGroup(groupID, senderID, model.ActionReadMessages, data)

	if sent != 1 {
		t.Errorf("expected 1 recipient, got %d", sent)
	}

	// Recipient should have received the message
	select {
	case msg := <-recipient.send:
		if string(msg) != string(data) {
			t.Errorf("expected %s, got %s", data, msg)
		}
	default:
		t.Error("expected recipient to receive message")
	}

	// Sender should NOT have received the message
	select {
	case <-sender.send:
		t.Error("sender should not receive own broadcast")
	default:
		// OK
	}

	// Outsider should NOT have received the message
	select {
	case <-outsider.send:
		t.Error("outsider should not receive group broadcast")
	default:
		// OK
	}
}

func TestHub_BroadcastToGroup_NoRecipients(t *testing.T) {
	hub := newTestHub()
	hub.permSvc = newTestPermSvc()
	groupID := uuid.New()
	data := []byte(`{"test":"empty"}`)

	sent := hub.broadcastToGroup(groupID, uuid.New(), model.ActionReadMessages, data)
	if sent != 0 {
		t.Errorf("expected 0 recipients, got %d", sent)
	}
}

// ---------------------------------------------------------------------------
// sendToUser
// ---------------------------------------------------------------------------

func TestHub_SendToUser(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	c1 := newTestClient(hub, userID, uuid.New(), nil)
	c2 := newTestClient(hub, userID, uuid.New(), nil)

	hub.addClient(c1)
	hub.addClient(c2)

	data := []byte(`{"test":"direct"}`)
	hub.sendToUser(userID, data)

	// Both clients should receive
	for _, c := range []*Client{c1, c2} {
		select {
		case msg := <-c.send:
			if string(msg) != string(data) {
				t.Errorf("expected %s, got %s", data, msg)
			}
		default:
			t.Error("expected client to receive message")
		}
	}
}

func TestHub_SendToUser_NoClients(t *testing.T) {
	hub := newTestHub()
	// Should not panic
	hub.sendToUser(uuid.New(), []byte(`test`))
}

// ---------------------------------------------------------------------------
// routeMessage
// ---------------------------------------------------------------------------

func TestHub_RouteMessage_LocationBroadcast(t *testing.T) {
	hub := newTestHub()
	hub.permSvc = newTestPermSvc()
	groupID := uuid.New()
	senderID := uuid.New()
	recipientID := uuid.New()

	recipient := newTestClient(hub, recipientID, uuid.New(), []uuid.UUID{groupID})
	hub.addClient(recipient)

	broadcast := service.LocationBroadcast{
		UserID:   senderID,
		GroupID:  groupID,
		Lat:      1.0,
		Lng:      2.0,
		Username: "sender",
	}
	payload, _ := json.Marshal(broadcast)

	hub.routeMessage(pubsub.Message{
		Channel: "group:" + groupID.String() + ":location",
		Payload: payload,
	})

	select {
	case msg := <-recipient.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("failed to unmarshal envelope: %v", err)
		}
		if env.Type != TypeLocationBroadcast {
			t.Errorf("expected type %s, got %s", TypeLocationBroadcast, env.Type)
		}
	default:
		t.Error("expected recipient to receive location broadcast")
	}
}

func TestHub_RouteMessage_GroupMessages(t *testing.T) {
	hub := newTestHub()
	hub.permSvc = newTestPermSvc()
	groupID := uuid.New()
	senderID := uuid.New()
	recipientID := uuid.New()

	recipient := newTestClient(hub, recipientID, uuid.New(), []uuid.UUID{groupID})
	hub.addClient(recipient)

	msgPayload, _ := json.Marshal(map[string]any{
		"sender_id": senderID,
		"content":   "hello",
	})

	hub.routeMessage(pubsub.Message{
		Channel: "group:" + groupID.String() + ":messages",
		Payload: msgPayload,
	})

	select {
	case msg := <-recipient.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("failed to unmarshal envelope: %v", err)
		}
		if env.Type != TypeMessageNew {
			t.Errorf("expected type %s, got %s", TypeMessageNew, env.Type)
		}
	default:
		t.Error("expected recipient to receive group message")
	}
}

func TestHub_RouteMessage_DirectMessage(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	client := newTestClient(hub, userID, uuid.New(), nil)
	hub.addClient(client)

	msgPayload, _ := json.Marshal(map[string]any{
		"sender_id": uuid.New(),
		"content":   "hi there",
	})

	hub.routeMessage(pubsub.Message{
		Channel: "user:" + userID.String() + ":direct",
		Payload: msgPayload,
	})

	select {
	case msg := <-client.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("failed to unmarshal envelope: %v", err)
		}
		if env.Type != TypeMessageNew {
			t.Errorf("expected type %s, got %s", TypeMessageNew, env.Type)
		}
	default:
		t.Error("expected client to receive direct message")
	}
}

func TestHub_RouteMessage_GroupDrawingUpdate(t *testing.T) {
	hub := newTestHub()
	hub.permSvc = newTestPermSvc()
	groupID := uuid.New()
	ownerID := uuid.New()
	recipientID := uuid.New()

	recipient := newTestClient(hub, recipientID, uuid.New(), []uuid.UUID{groupID})
	hub.addClient(recipient)

	drawingPayload, _ := json.Marshal(map[string]any{
		"owner_id": ownerID,
		"name":     "my drawing",
	})

	hub.routeMessage(pubsub.Message{
		Channel: "group:" + groupID.String() + ":drawings",
		Payload: drawingPayload,
	})

	select {
	case msg := <-recipient.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("failed to unmarshal envelope: %v", err)
		}
		if env.Type != TypeDrawingUpdated {
			t.Errorf("expected type %s, got %s", TypeDrawingUpdated, env.Type)
		}
	default:
		t.Error("expected recipient to receive drawing update")
	}
}

func TestHub_RouteMessage_UserDrawingUpdate(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	client := newTestClient(hub, userID, uuid.New(), nil)
	hub.addClient(client)

	drawingPayload, _ := json.Marshal(map[string]any{
		"owner_id": uuid.New(),
		"name":     "shared drawing",
	})

	hub.routeMessage(pubsub.Message{
		Channel: "user:" + userID.String() + ":drawings",
		Payload: drawingPayload,
	})

	select {
	case msg := <-client.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("failed to unmarshal envelope: %v", err)
		}
		if env.Type != TypeDrawingUpdated {
			t.Errorf("expected type %s, got %s", TypeDrawingUpdated, env.Type)
		}
	default:
		t.Error("expected client to receive user drawing update")
	}
}

func TestHub_RouteMessage_InvalidChannel(t *testing.T) {
	hub := newTestHub()
	// Should not panic
	hub.routeMessage(pubsub.Message{
		Channel: "unknown:channel",
		Payload: []byte(`{}`),
	})
}

func TestHub_RouteMessage_InvalidPayload(t *testing.T) {
	hub := newTestHub()
	hub.permSvc = newTestPermSvc()
	groupID := uuid.New()
	// Invalid JSON — should not panic
	hub.routeMessage(pubsub.Message{
		Channel: "group:" + groupID.String() + ":location",
		Payload: []byte(`not-json`),
	})
}

// ---------------------------------------------------------------------------
// SendConnected
// ---------------------------------------------------------------------------

func TestHub_SendConnected(t *testing.T) {
	groupID := uuid.New()
	groupRepo := &mock.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			if id == groupID {
				return &model.Group{ID: groupID, Name: "Test Group"}, nil
			}
			return nil, model.ErrNotFound("group")
		},
	}

	hub := &Hub{
		clients:   make(map[uuid.UUID]map[*Client]struct{}),
		groupRepo: groupRepo,
	}

	userID := uuid.New()
	client := newTestClient(hub, userID, uuid.New(), []uuid.UUID{groupID})
	client.hub = hub

	hub.SendConnected(context.Background(), client)

	select {
	case msg := <-client.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("failed to unmarshal envelope: %v", err)
		}
		if env.Type != TypeConnected {
			t.Errorf("expected type %s, got %s", TypeConnected, env.Type)
		}
		var payload ConnectedPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		if payload.UserID != userID {
			t.Errorf("expected user ID %s, got %s", userID, payload.UserID)
		}
		if len(payload.Groups) != 1 || payload.Groups[0].Name != "Test Group" {
			t.Errorf("unexpected groups: %+v", payload.Groups)
		}
	default:
		t.Error("expected client to receive connected message")
	}
}

// ---------------------------------------------------------------------------
// SendSnapshot
// ---------------------------------------------------------------------------

func TestHub_SendSnapshot(t *testing.T) {
	groupID := uuid.New()
	userID := uuid.New()
	deviceID := uuid.New()
	otherUserID := uuid.New()
	otherDeviceID := uuid.New()
	now := time.Now()
	otherDisplayName := "Other User"
	otherDeviceName := "Phone"

	locationRepo := &mock.LocationRepo{
		GetLatestByGroupFn: func(ctx context.Context, gid uuid.UUID) ([]repository.LocationRecord, error) {
			if gid == groupID {
				return []repository.LocationRecord{
					{
						UserID:     userID,
						DeviceID:   deviceID, // Same as client — should be excluded
						Lat:        1.0,
						Lng:        2.0,
						RecordedAt: now,
					},
					{
						UserID:      otherUserID,
						DeviceID:    otherDeviceID,
						Username:    "other",
						DisplayName: &otherDisplayName,
						DeviceName:  &otherDeviceName,
						IsPrimary:   true,
						Lat:         3.0,
						Lng:         4.0,
						RecordedAt:  now,
					},
				}, nil
			}
			return nil, nil
		},
	}

	// Need a LocationService — construct with minimal deps
	ps := pubsub.NewMockPubSub()
	groupRepo := &mock.GroupRepo{}
	locationSvc := service.NewLocationService(locationRepo, groupRepo, ps, time.Second)

	hub := &Hub{
		clients:     make(map[uuid.UUID]map[*Client]struct{}),
		locationSvc: locationSvc,
		permSvc:     newTestPermSvc(),
	}

	client := newTestClient(hub, userID, deviceID, []uuid.UUID{groupID})
	client.hub = hub

	hub.SendSnapshot(context.Background(), client)

	select {
	case msg := <-client.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("failed to unmarshal envelope: %v", err)
		}
		if env.Type != TypeLocationSnapshot {
			t.Errorf("expected type %s, got %s", TypeLocationSnapshot, env.Type)
		}
		var payload LocationSnapshotPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		if payload.GroupID != groupID {
			t.Errorf("expected group ID %s, got %s", groupID, payload.GroupID)
		}
		// Should only include the other user's location (own device excluded)
		if len(payload.Locations) != 1 {
			t.Fatalf("expected 1 location, got %d", len(payload.Locations))
		}
		if payload.Locations[0].UserID != otherUserID {
			t.Errorf("expected other user, got %s", payload.Locations[0].UserID)
		}
		if payload.Locations[0].Lat != 3.0 || payload.Locations[0].Lng != 4.0 {
			t.Errorf("unexpected coordinates: %f, %f", payload.Locations[0].Lat, payload.Locations[0].Lng)
		}
		if payload.Locations[0].DisplayName != "Other User" {
			t.Errorf("expected display name 'Other User', got %s", payload.Locations[0].DisplayName)
		}
	default:
		t.Error("expected client to receive snapshot message")
	}
}

func TestHub_SendSnapshot_NoLocations(t *testing.T) {
	groupID := uuid.New()
	locationRepo := &mock.LocationRepo{
		GetLatestByGroupFn: func(ctx context.Context, gid uuid.UUID) ([]repository.LocationRecord, error) {
			return nil, nil
		},
	}

	ps := pubsub.NewMockPubSub()
	locationSvc := service.NewLocationService(locationRepo, &mock.GroupRepo{}, ps, time.Second)

	hub := &Hub{
		clients:     make(map[uuid.UUID]map[*Client]struct{}),
		locationSvc: locationSvc,
		permSvc:     newTestPermSvc(),
	}

	client := newTestClient(hub, uuid.New(), uuid.New(), []uuid.UUID{groupID})
	client.hub = hub

	hub.SendSnapshot(context.Background(), client)

	// No locations → no snapshot message sent
	select {
	case <-client.send:
		t.Error("expected no snapshot when there are no locations")
	default:
		// OK
	}
}

// ---------------------------------------------------------------------------
// handleLocationUpdate
// ---------------------------------------------------------------------------

func TestHub_HandleLocationUpdate_DeviceMismatch(t *testing.T) {
	hub := newTestHub()
	hub.permSvc = newTestPermSvc()
	clientDeviceID := uuid.New()
	client := newTestClient(hub, uuid.New(), clientDeviceID, nil)

	wrongDeviceID := uuid.New()
	payload := &LocationUpdatePayload{
		DeviceID: wrongDeviceID.String(),
		Lat:      1.0,
		Lng:      2.0,
	}

	hub.handleLocationUpdate(context.Background(), client, payload)

	// Should send an error message
	select {
	case msg := <-client.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if env.Type != TypeError {
			t.Errorf("expected error type, got %s", env.Type)
		}
	default:
		t.Error("expected error message for device mismatch")
	}
}

func TestHub_HandleLocationUpdate_InvalidDeviceID(t *testing.T) {
	hub := newTestHub()
	hub.permSvc = newTestPermSvc()
	client := newTestClient(hub, uuid.New(), uuid.New(), nil)

	payload := &LocationUpdatePayload{
		DeviceID: "not-a-uuid",
		Lat:      1.0,
		Lng:      2.0,
	}

	hub.handleLocationUpdate(context.Background(), client, payload)

	select {
	case msg := <-client.send:
		var env Envelope
		json.Unmarshal(msg, &env)
		if env.Type != TypeError {
			t.Errorf("expected error type, got %s", env.Type)
		}
	default:
		t.Error("expected error message for invalid device ID")
	}
}

// ---------------------------------------------------------------------------
// Client helper tests
// ---------------------------------------------------------------------------

func TestClient_SendError(t *testing.T) {
	hub := newTestHub()
	client := newTestClient(hub, uuid.New(), uuid.New(), nil)

	client.sendError("test error")

	select {
	case msg := <-client.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if env.Type != TypeError {
			t.Errorf("expected type %s, got %s", TypeError, env.Type)
		}
		var payload ErrorPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.Message != "test error" {
			t.Errorf("expected 'test error', got %s", payload.Message)
		}
	default:
		t.Error("expected error message in send channel")
	}
}

func TestClient_SendJSON(t *testing.T) {
	hub := newTestHub()
	client := newTestClient(hub, uuid.New(), uuid.New(), nil)

	client.sendJSON(TypeConnected, ConnectedPayload{
		UserID: client.userID,
		Groups: nil,
	})

	select {
	case msg := <-client.send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if env.Type != TypeConnected {
			t.Errorf("expected type %s, got %s", TypeConnected, env.Type)
		}
	default:
		t.Error("expected message in send channel")
	}
}
