package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Device represents a user's device.
type Device struct {
	ID         uuid.UUID  `json:"-"`
	UserID     uuid.UUID  `json:"-"`
	Name       string     `json:"-"`
	DeviceType string     `json:"-"`
	DeviceUID  *string    `json:"-"`
	UserAgent  *string    `json:"-"`
	IsPrimary  bool       `json:"-"`
	LastSeenAt *time.Time `json:"-"`
	CreatedAt  time.Time  `json:"-"`
	UpdatedAt  time.Time  `json:"-"`
}

// DeviceResponse is the JSON representation of a device.
type DeviceResponse struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Name       string     `json:"name"`
	DeviceType string     `json:"device_type"`
	DeviceUID  string     `json:"device_uid,omitempty"`
	UserAgent  string     `json:"user_agent,omitempty"`
	IsPrimary  bool       `json:"is_primary"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ToResponse converts a Device to its API response representation.
func (d *Device) ToResponse() DeviceResponse {
	uid := ""
	if d.DeviceUID != nil {
		uid = *d.DeviceUID
	}
	ua := ""
	if d.UserAgent != nil {
		ua = *d.UserAgent
	}
	return DeviceResponse{
		ID:         d.ID,
		UserID:     d.UserID,
		Name:       d.Name,
		DeviceType: d.DeviceType,
		DeviceUID:  uid,
		UserAgent:  ua,
		IsPrimary:  d.IsPrimary,
		LastSeenAt: d.LastSeenAt,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

// CreateDeviceRequest is the expected body for registering a new device.
type CreateDeviceRequest struct {
	Name       string `json:"name"`
	DeviceType string `json:"device_type"`
}

// Validate checks that required fields are present.
func (r *CreateDeviceRequest) Validate() error {
	if r.Name == "" {
		return ErrValidation("name is required")
	}
	if r.DeviceType == "" {
		r.DeviceType = "web"
	}
	validTypes := map[string]bool{"web": true, "ios": true, "android": true, "cli": true}
	if !validTypes[r.DeviceType] {
		return ErrValidation("device_type must be web, ios, or android")
	}
	return nil
}

// DeviceResolveResponse is returned by the resolve endpoint.
// When Matched is true, Device contains the recognised device.
// When Matched is false, ExistingDevices lists the user's registered devices
// so the client can prompt the user to pick one or create a new device.
type DeviceResolveResponse struct {
	Matched         bool             `json:"matched"`
	Device          *DeviceResponse  `json:"device,omitempty"`
	ExistingDevices []DeviceResponse `json:"existing_devices,omitempty"`
}

// UpdateDeviceRequest is the expected body for updating a device.
type UpdateDeviceRequest struct {
	Name *string `json:"name"`
}

// Validate checks that the supplied fields are valid.
func (r *UpdateDeviceRequest) Validate() error {
	if r.Name != nil {
		trimmed := strings.TrimSpace(*r.Name)
		if trimmed == "" {
			return ErrValidation("name must not be empty")
		}
		if len(trimmed) > 50 {
			return ErrValidation("name must not exceed 50 characters")
		}
		*r.Name = trimmed
	}
	return nil
}
