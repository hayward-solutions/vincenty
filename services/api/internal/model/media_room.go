package model

import (
	"time"

	"github.com/google/uuid"
)

// RoomType constants for media rooms.
const (
	RoomTypeStream = "stream"
	RoomTypePTT    = "ptt_channel"
)

// ParticipantRole constants.
const (
	ParticipantRoleHost        = "host"
	ParticipantRoleParticipant = "participant"
	ParticipantRoleViewer      = "viewer"
)

// MediaRoom represents an active or historical video/voice/PTT room.
type MediaRoom struct {
	ID              uuid.UUID  `json:"-"`
	Name            string     `json:"-"`
	RoomType        string     `json:"-"`
	GroupID         *uuid.UUID `json:"-"`
	CreatedBy       uuid.UUID  `json:"-"`
	LiveKitRoom     string     `json:"-"`
	IsActive        bool       `json:"-"`
	MaxParticipants int        `json:"-"`
	Metadata        []byte     `json:"-"`
	CreatedAt       time.Time  `json:"-"`
	EndedAt         *time.Time `json:"-"`
}

// MediaRoomResponse is the JSON representation of a media room returned by the API.
type MediaRoomResponse struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	RoomType        string     `json:"room_type"`
	GroupID         *uuid.UUID `json:"group_id,omitempty"`
	CreatedBy       uuid.UUID  `json:"created_by"`
	LiveKitRoom     string     `json:"livekit_room"`
	IsActive        bool       `json:"is_active"`
	MaxParticipants int        `json:"max_participants"`
	CreatedAt       time.Time  `json:"created_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
}

// ToResponse converts a MediaRoom to its API response representation.
func (r *MediaRoom) ToResponse() MediaRoomResponse {
	return MediaRoomResponse{
		ID:              r.ID,
		Name:            r.Name,
		RoomType:        r.RoomType,
		GroupID:         r.GroupID,
		CreatedBy:       r.CreatedBy,
		LiveKitRoom:     r.LiveKitRoom,
		IsActive:        r.IsActive,
		MaxParticipants: r.MaxParticipants,
		CreatedAt:       r.CreatedAt,
		EndedAt:         r.EndedAt,
	}
}

// MediaRoomParticipant represents a user's participation in a media room.
type MediaRoomParticipant struct {
	ID       uuid.UUID  `json:"-"`
	RoomID   uuid.UUID  `json:"-"`
	UserID   uuid.UUID  `json:"-"`
	DeviceID *uuid.UUID `json:"-"`
	Role     string     `json:"-"`
	JoinedAt time.Time  `json:"-"`
	LeftAt   *time.Time `json:"-"`
}

// MediaRoomParticipantResponse is the JSON representation of a room participant.
type MediaRoomParticipantResponse struct {
	ID       uuid.UUID  `json:"id"`
	RoomID   uuid.UUID  `json:"room_id"`
	UserID   uuid.UUID  `json:"user_id"`
	DeviceID *uuid.UUID `json:"device_id,omitempty"`
	Role     string     `json:"role"`
	JoinedAt time.Time  `json:"joined_at"`
	LeftAt   *time.Time `json:"left_at,omitempty"`
}

// ToResponse converts a MediaRoomParticipant to its API response representation.
func (p *MediaRoomParticipant) ToResponse() MediaRoomParticipantResponse {
	return MediaRoomParticipantResponse{
		ID:       p.ID,
		RoomID:   p.RoomID,
		UserID:   p.UserID,
		DeviceID: p.DeviceID,
		Role:     p.Role,
		JoinedAt: p.JoinedAt,
		LeftAt:   p.LeftAt,
	}
}

// JoinRoomResponse is returned when a user joins a media room.
type JoinRoomResponse struct {
	Room  MediaRoomResponse `json:"room"`
	Token string            `json:"token"`
	URL   string            `json:"url"`
}
