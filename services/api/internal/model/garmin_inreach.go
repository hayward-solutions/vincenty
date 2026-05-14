package model

import (
	"time"

	"github.com/google/uuid"
)

// GarminInReachFeed represents a configured Garmin InReach MapShare feed
// linked to a Vincenty user and device.
type GarminInReachFeed struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	DeviceID     uuid.UUID
	MapShareID   string
	FeedPassword *string
	PollInterval time.Duration
	Enabled      bool
	LastPolledAt *time.Time
	LastPointAt  *time.Time
	ErrorCount   int
	LastError    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// GarminInReachFeedResponse is the JSON representation returned by the API.
type GarminInReachFeedResponse struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	DeviceID     uuid.UUID  `json:"device_id"`
	MapShareID   string     `json:"mapshare_id"`
	PollInterval string     `json:"poll_interval"`
	Enabled      bool       `json:"enabled"`
	LastPolledAt *time.Time `json:"last_polled_at,omitempty"`
	LastPointAt  *time.Time `json:"last_point_at,omitempty"`
	ErrorCount   int        `json:"error_count"`
	LastError    *string    `json:"last_error,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ToResponse converts a GarminInReachFeed to its API response representation.
func (f *GarminInReachFeed) ToResponse() GarminInReachFeedResponse {
	return GarminInReachFeedResponse{
		ID:           f.ID,
		UserID:       f.UserID,
		DeviceID:     f.DeviceID,
		MapShareID:   f.MapShareID,
		PollInterval: f.PollInterval.String(),
		Enabled:      f.Enabled,
		LastPolledAt: f.LastPolledAt,
		LastPointAt:  f.LastPointAt,
		ErrorCount:   f.ErrorCount,
		LastError:    f.LastError,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}

// CreateGarminInReachFeedRequest is the expected body for creating a feed.
type CreateGarminInReachFeedRequest struct {
	UserID       uuid.UUID `json:"user_id"`
	MapShareID   string    `json:"mapshare_id"`
	FeedPassword *string   `json:"feed_password,omitempty"`
	PollInterval string    `json:"poll_interval"`
}

// Validate checks that required fields are present.
func (r *CreateGarminInReachFeedRequest) Validate() error {
	if r.UserID == uuid.Nil {
		return ErrValidation("user_id is required")
	}
	if r.MapShareID == "" {
		return ErrValidation("mapshare_id is required")
	}
	if r.PollInterval == "" {
		r.PollInterval = "2m"
	}
	d, err := time.ParseDuration(r.PollInterval)
	if err != nil {
		return ErrValidation("poll_interval must be a valid duration (e.g. 2m, 120s)")
	}
	if d < 30*time.Second {
		return ErrValidation("poll_interval must be at least 30s")
	}
	return nil
}

// UpdateGarminInReachFeedRequest is the expected body for updating a feed.
type UpdateGarminInReachFeedRequest struct {
	FeedPassword *string `json:"feed_password,omitempty"`
	PollInterval *string `json:"poll_interval,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
}

// Validate checks that the supplied fields are valid.
func (r *UpdateGarminInReachFeedRequest) Validate() error {
	if r.PollInterval != nil {
		d, err := time.ParseDuration(*r.PollInterval)
		if err != nil {
			return ErrValidation("poll_interval must be a valid duration (e.g. 2m, 120s)")
		}
		if d < 30*time.Second {
			return ErrValidation("poll_interval must be at least 30s")
		}
	}
	return nil
}
