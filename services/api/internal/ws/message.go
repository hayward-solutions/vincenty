package ws

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Message type constants.
const (
	// Client → Server
	TypeLocationUpdate = "location_update"

	// Server → Client
	TypeLocationBroadcast = "location_broadcast"
	TypeLocationSnapshot  = "location_snapshot"
	TypeMessageNew        = "message_new"
	TypeDrawingUpdated    = "drawing_updated"
	TypeConnected         = "connected"
	TypeError             = "error"

	// Streams
	TypeStreamStarted = "stream_started"
	TypeStreamStopped = "stream_stopped"

	// Push-to-Talk (Client → Server)
	TypePTTFloorRequest = "ptt_floor_request"
	TypePTTFloorRelease = "ptt_floor_release"

	// Push-to-Talk (Server → Client)
	TypePTTFloorGranted  = "ptt_floor_granted"
	TypePTTFloorReleased = "ptt_floor_released"
)

// Envelope is the outer JSON wrapper for all WebSocket messages.
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// NewEnvelope creates an Envelope with a marshalled payload.
func NewEnvelope(msgType string, payload any) ([]byte, error) {
	p, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Envelope{Type: msgType, Payload: p})
}

// --------------------------------------------------------------------------
// Client → Server payloads
// --------------------------------------------------------------------------

// LocationUpdatePayload is sent by the client to report its position.
type LocationUpdatePayload struct {
	DeviceID string   `json:"device_id"`
	Lat      float64  `json:"lat"`
	Lng      float64  `json:"lng"`
	Altitude *float64 `json:"altitude,omitempty"`
	Heading  *float64 `json:"heading,omitempty"`
	Speed    *float64 `json:"speed,omitempty"`
	Accuracy *float64 `json:"accuracy,omitempty"`
}

// --------------------------------------------------------------------------
// Server → Client payloads
// --------------------------------------------------------------------------

// LocationBroadcastPayload is sent to group members when a device's location changes.
type LocationBroadcastPayload struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	DeviceID    uuid.UUID `json:"device_id"`
	DeviceName  string    `json:"device_name"`
	IsPrimary   bool      `json:"is_primary"`
	GroupID     uuid.UUID `json:"group_id"`
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	Altitude    *float64  `json:"altitude,omitempty"`
	Heading     *float64  `json:"heading,omitempty"`
	Speed       *float64  `json:"speed,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// LocationSnapshotPayload is sent on connect with the latest positions for a group.
type LocationSnapshotPayload struct {
	GroupID   uuid.UUID                  `json:"group_id"`
	Locations []LocationBroadcastPayload `json:"locations"`
}

// ConnectedPayload is sent after a successful WebSocket handshake.
type ConnectedPayload struct {
	UserID uuid.UUID        `json:"user_id"`
	Groups []ConnectedGroup `json:"groups"`
}

// ConnectedGroup is a summary of a group the user belongs to.
type ConnectedGroup struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ErrorPayload is sent when the server encounters an error.
type ErrorPayload struct {
	Message string `json:"message"`
}

// PTTFloorRequestPayload is sent by the client to request or release the PTT floor.
type PTTFloorRequestPayload struct {
	ChannelID uuid.UUID `json:"channel_id"`
}
