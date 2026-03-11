package model

import (
	"time"

	"github.com/google/uuid"
)

// PTTChannel represents a persistent push-to-talk audio channel for a group.
type PTTChannel struct {
	ID        uuid.UUID `json:"-"`
	GroupID   uuid.UUID `json:"-"`
	RoomID    uuid.UUID `json:"-"`
	Name      string    `json:"-"`
	IsDefault bool      `json:"-"`
	CreatedAt time.Time `json:"-"`
}

// PTTChannelResponse is the JSON representation of a PTT channel returned by the API.
type PTTChannelResponse struct {
	ID        uuid.UUID `json:"id"`
	GroupID   uuid.UUID `json:"group_id"`
	RoomID    uuid.UUID `json:"room_id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a PTTChannel to its API response representation.
func (c *PTTChannel) ToResponse() PTTChannelResponse {
	return PTTChannelResponse{
		ID:        c.ID,
		GroupID:   c.GroupID,
		RoomID:    c.RoomID,
		Name:      c.Name,
		IsDefault: c.IsDefault,
		CreatedAt: c.CreatedAt,
	}
}

// CreatePTTChannelRequest is the expected body for creating a PTT channel.
type CreatePTTChannelRequest struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// Validate checks that required fields are present.
func (r *CreatePTTChannelRequest) Validate() error {
	if r.Name == "" {
		return ErrValidation("name is required")
	}
	if len(r.Name) > 255 {
		return ErrValidation("name must be 255 characters or less")
	}
	return nil
}

// JoinPTTChannelResponse is returned when a user joins a PTT channel.
type JoinPTTChannelResponse struct {
	Channel PTTChannelResponse `json:"channel"`
	Token   string             `json:"token"`
	URL     string             `json:"url"`
}
