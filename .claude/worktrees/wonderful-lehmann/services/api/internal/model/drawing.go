package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Drawing represents a map drawing (annotation) in the system.
type Drawing struct {
	ID        uuid.UUID       `json:"-"`
	OwnerID   uuid.UUID       `json:"-"`
	Name      string          `json:"-"`
	GeoJSON   json.RawMessage `json:"-"`
	CreatedAt time.Time       `json:"-"`
	UpdatedAt time.Time       `json:"-"`
}

// DrawingResponse is the JSON representation returned by the API.
type DrawingResponse struct {
	ID          uuid.UUID       `json:"id"`
	OwnerID     uuid.UUID       `json:"owner_id"`
	Username    string          `json:"username"`
	DisplayName string          `json:"display_name"`
	Name        string          `json:"name"`
	GeoJSON     json.RawMessage `json:"geojson"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// DrawingShareInfo describes a single share target for a drawing.
type DrawingShareInfo struct {
	Type      string    `json:"type"` // "group" or "user"
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	SharedAt  time.Time `json:"shared_at"`
	MessageID uuid.UUID `json:"message_id"`
}

// DrawingWithUser is a join result containing drawing + owner details.
type DrawingWithUser struct {
	Drawing
	Username    string
	DisplayName *string
}

// ToResponse converts a DrawingWithUser to its API response.
func (d *DrawingWithUser) ToResponse() DrawingResponse {
	displayName := ""
	if d.DisplayName != nil {
		displayName = *d.DisplayName
	}

	return DrawingResponse{
		ID:          d.ID,
		OwnerID:     d.OwnerID,
		Username:    d.Username,
		DisplayName: displayName,
		Name:        d.Name,
		GeoJSON:     d.GeoJSON,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
