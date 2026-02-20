package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Message represents a message in the system.
type Message struct {
	ID             uuid.UUID        `json:"-"`
	SenderID       uuid.UUID        `json:"-"`
	SenderDeviceID *uuid.UUID       `json:"-"`
	GroupID        *uuid.UUID       `json:"-"`
	RecipientID    *uuid.UUID       `json:"-"`
	Content        *string          `json:"-"`
	MessageType    string           `json:"-"`
	Lat            *float64         `json:"-"` // extracted from PostGIS location
	Lng            *float64         `json:"-"`
	Metadata       *json.RawMessage `json:"-"`
	CreatedAt      time.Time        `json:"-"`
}

// Attachment represents a file attached to a message.
type Attachment struct {
	ID          uuid.UUID `json:"-"`
	MessageID   uuid.UUID `json:"-"`
	Filename    string    `json:"-"`
	ContentType string    `json:"-"`
	SizeBytes   int64     `json:"-"`
	ObjectKey   string    `json:"-"`
	CreatedAt   time.Time `json:"-"`
}

// MessageResponse is the JSON representation returned by the API.
type MessageResponse struct {
	ID          uuid.UUID            `json:"id"`
	SenderID    uuid.UUID            `json:"sender_id"`
	Username    string               `json:"username"`
	DisplayName string               `json:"display_name"`
	GroupID     *uuid.UUID           `json:"group_id,omitempty"`
	RecipientID *uuid.UUID           `json:"recipient_id,omitempty"`
	Content     string               `json:"content"`
	MessageType string               `json:"message_type"`
	Lat         *float64             `json:"lat,omitempty"`
	Lng         *float64             `json:"lng,omitempty"`
	Metadata    *json.RawMessage     `json:"metadata,omitempty"`
	Attachments []AttachmentResponse `json:"attachments"`
	CreatedAt   time.Time            `json:"created_at"`
}

// AttachmentResponse is the JSON representation of an attachment.
type AttachmentResponse struct {
	ID          uuid.UUID `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

// ToResponse converts an Attachment to its API response.
func (a *Attachment) ToResponse() AttachmentResponse {
	return AttachmentResponse{
		ID:          a.ID,
		Filename:    a.Filename,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		CreatedAt:   a.CreatedAt,
	}
}

// MessageWithUser is a join result containing message + sender details.
type MessageWithUser struct {
	Message
	Username    string
	DisplayName *string
	Attachments []Attachment
}

// ToResponse converts a MessageWithUser to its API response.
func (m *MessageWithUser) ToResponse() MessageResponse {
	content := ""
	if m.Content != nil {
		content = *m.Content
	}
	displayName := ""
	if m.DisplayName != nil {
		displayName = *m.DisplayName
	}

	atts := make([]AttachmentResponse, len(m.Attachments))
	for i, a := range m.Attachments {
		atts[i] = a.ToResponse()
	}

	return MessageResponse{
		ID:          m.ID,
		SenderID:    m.SenderID,
		Username:    m.Username,
		DisplayName: displayName,
		GroupID:     m.GroupID,
		RecipientID: m.RecipientID,
		Content:     content,
		MessageType: m.MessageType,
		Lat:         m.Lat,
		Lng:         m.Lng,
		Metadata:    m.Metadata,
		Attachments: atts,
		CreatedAt:   m.CreatedAt,
	}
}
