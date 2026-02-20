package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLog represents a single audit trail entry.
type AuditLog struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	DeviceID     *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	GroupID      *uuid.UUID
	Metadata     *json.RawMessage
	Lat          *float64
	Lng          *float64
	IPAddress    string
	CreatedAt    time.Time
}

// AuditLogWithUser is a join result containing the audit entry + user details.
type AuditLogWithUser struct {
	AuditLog
	Username    string
	DisplayName *string
}

// AuditLogResponse is the JSON representation returned by the API.
type AuditLogResponse struct {
	ID           uuid.UUID        `json:"id"`
	UserID       uuid.UUID        `json:"user_id"`
	Username     string           `json:"username"`
	DisplayName  string           `json:"display_name"`
	DeviceID     *uuid.UUID       `json:"device_id,omitempty"`
	Action       string           `json:"action"`
	ResourceType string           `json:"resource_type"`
	ResourceID   *uuid.UUID       `json:"resource_id,omitempty"`
	GroupID      *uuid.UUID       `json:"group_id,omitempty"`
	Metadata     *json.RawMessage `json:"metadata,omitempty"`
	Lat          *float64         `json:"lat,omitempty"`
	Lng          *float64         `json:"lng,omitempty"`
	IPAddress    string           `json:"ip_address"`
	CreatedAt    time.Time        `json:"created_at"`
}

// ToResponse converts an AuditLogWithUser to its API response.
func (a *AuditLogWithUser) ToResponse() AuditLogResponse {
	dn := ""
	if a.DisplayName != nil {
		dn = *a.DisplayName
	}
	return AuditLogResponse{
		ID:           a.ID,
		UserID:       a.UserID,
		Username:     a.Username,
		DisplayName:  dn,
		DeviceID:     a.DeviceID,
		Action:       a.Action,
		ResourceType: a.ResourceType,
		ResourceID:   a.ResourceID,
		GroupID:      a.GroupID,
		Metadata:     a.Metadata,
		Lat:          a.Lat,
		Lng:          a.Lng,
		IPAddress:    a.IPAddress,
		CreatedAt:    a.CreatedAt,
	}
}

// AuditFilters holds query parameters for filtering audit logs.
type AuditFilters struct {
	From         *time.Time
	To           *time.Time
	Action       string
	ResourceType string
	Page         int
	PageSize     int
}
