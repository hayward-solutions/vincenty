package model

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Stream
// ---------------------------------------------------------------------------

// Stream represents a live or recorded video stream.
type Stream struct {
	ID            uuid.UUID  `json:"-"`
	Title         string     `json:"-"`
	BroadcasterID *uuid.UUID `json:"-"`
	StreamKeyID   *uuid.UUID `json:"-"`
	SourceType    string     `json:"-"` // "browser", "rtsp", "rtmp"
	Status        string     `json:"-"` // "live", "ended"
	MediaPath     string     `json:"-"`
	RecordingURL  *string    `json:"-"`
	StartedAt     time.Time  `json:"-"`
	EndedAt       *time.Time `json:"-"`
	CreatedAt     time.Time  `json:"-"`
	UpdatedAt     time.Time  `json:"-"`
}

// StreamWithDetails is a join result containing stream + broadcaster details + groups.
type StreamWithDetails struct {
	Stream
	Username    *string
	DisplayName *string
	Groups      []uuid.UUID
}

// ToResponse converts a StreamWithDetails to its API response.
func (s *StreamWithDetails) ToResponse() StreamResponse {
	username := ""
	if s.Username != nil {
		username = *s.Username
	}
	displayName := ""
	if s.DisplayName != nil {
		displayName = *s.DisplayName
	}
	groups := s.Groups
	if groups == nil {
		groups = []uuid.UUID{}
	}

	return StreamResponse{
		ID:            s.ID,
		Title:         s.Title,
		BroadcasterID: s.BroadcasterID,
		Username:      username,
		DisplayName:   displayName,
		SourceType:    s.SourceType,
		Status:        s.Status,
		MediaPath:     s.MediaPath,
		RecordingURL:  s.RecordingURL,
		Groups:        groups,
		StartedAt:     s.StartedAt,
		EndedAt:       s.EndedAt,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

// StreamResponse is the JSON representation returned by the API.
type StreamResponse struct {
	ID            uuid.UUID   `json:"id"`
	Title         string      `json:"title"`
	BroadcasterID *uuid.UUID  `json:"broadcaster_id,omitempty"`
	Username      string      `json:"username,omitempty"`
	DisplayName   string      `json:"display_name,omitempty"`
	SourceType    string      `json:"source_type"`
	Status        string      `json:"status"`
	MediaPath     string      `json:"media_path"`
	RecordingURL  *string     `json:"recording_url,omitempty"`
	Groups        []uuid.UUID `json:"groups"`
	StartedAt     time.Time   `json:"started_at"`
	EndedAt       *time.Time  `json:"ended_at,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// Stream Key (hardware device authentication)
// ---------------------------------------------------------------------------

// StreamKey represents a pre-shared key for hardware device streaming.
type StreamKey struct {
	ID        uuid.UUID `json:"-"`
	Label     string    `json:"-"`
	KeyHash   string    `json:"-"`
	CreatedBy uuid.UUID `json:"-"`
	IsActive  bool      `json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// StreamKeyWithGroups includes the default group IDs.
type StreamKeyWithGroups struct {
	StreamKey
	GroupIDs []uuid.UUID
}

// ToResponse converts a StreamKeyWithGroups to its API response.
func (k *StreamKeyWithGroups) ToResponse() StreamKeyResponse {
	groupIDs := k.GroupIDs
	if groupIDs == nil {
		groupIDs = []uuid.UUID{}
	}
	return StreamKeyResponse{
		ID:        k.ID,
		Label:     k.Label,
		GroupIDs:  groupIDs,
		IsActive:  k.IsActive,
		CreatedAt: k.CreatedAt,
		UpdatedAt: k.UpdatedAt,
	}
}

// StreamKeyResponse is the JSON representation returned by the API.
type StreamKeyResponse struct {
	ID        uuid.UUID   `json:"id"`
	Label     string      `json:"label"`
	GroupIDs  []uuid.UUID `json:"group_ids"`
	IsActive  bool        `json:"is_active"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	// Key is the plaintext key, only returned once on creation.
	Key string `json:"key,omitempty"`
}

// ---------------------------------------------------------------------------
// Stream Location (GPS telemetry for map-synced playback)
// ---------------------------------------------------------------------------

// StreamLocation represents a single GPS point recorded during a stream.
type StreamLocation struct {
	ID         uuid.UUID `json:"-"`
	StreamID   uuid.UUID `json:"-"`
	Lat        float64   `json:"-"`
	Lng        float64   `json:"-"`
	Altitude   *float64  `json:"-"`
	Heading    *float64  `json:"-"`
	Speed      *float64  `json:"-"`
	RecordedAt time.Time `json:"-"`
}

// StreamLocationResponse is the JSON representation for GPS telemetry.
type StreamLocationResponse struct {
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Altitude   *float64  `json:"altitude,omitempty"`
	Heading    *float64  `json:"heading,omitempty"`
	Speed      *float64  `json:"speed,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
}
