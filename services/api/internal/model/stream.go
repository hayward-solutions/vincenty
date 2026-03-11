package model

import (
	"time"

	"github.com/google/uuid"
)

// SourceType constants for streams.
const (
	SourceTypeRTSP         = "rtsp"
	SourceTypeRTMP         = "rtmp"
	SourceTypeWHIP         = "whip"
	SourceTypeDeviceCamera = "device_camera"
	SourceTypeScreenShare  = "screen_share"
)

// Stream represents a media stream (camera, RTSP feed, screen share, etc.).
type Stream struct {
	ID               uuid.UUID `json:"-"`
	Name             string    `json:"-"`
	SourceType       string    `json:"-"`
	SourceURL        *string   `json:"-"`
	GroupID          uuid.UUID `json:"-"`
	CreatedBy        uuid.UUID `json:"-"`
	LiveKitIngressID *string   `json:"-"`
	LiveKitRoom      *string   `json:"-"`
	StreamKey        *string   `json:"-"`
	IsActive         bool      `json:"-"`
	Metadata         []byte    `json:"-"`
	CreatedAt        time.Time `json:"-"`
	UpdatedAt        time.Time `json:"-"`
}

// StreamResponse is the JSON representation of a stream returned by the API.
type StreamResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	SourceType string    `json:"source_type"`
	SourceURL  string    `json:"source_url,omitempty"`
	GroupID    uuid.UUID `json:"group_id"`
	CreatedBy  uuid.UUID `json:"created_by"`
	StreamKey  string    `json:"stream_key,omitempty"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ToResponse converts a Stream to its API response representation.
func (s *Stream) ToResponse() StreamResponse {
	sourceURL := ""
	if s.SourceURL != nil {
		sourceURL = *s.SourceURL
	}
	streamKey := ""
	if s.StreamKey != nil {
		streamKey = *s.StreamKey
	}
	return StreamResponse{
		ID:         s.ID,
		Name:       s.Name,
		SourceType: s.SourceType,
		SourceURL:  sourceURL,
		GroupID:    s.GroupID,
		CreatedBy:  s.CreatedBy,
		StreamKey:  streamKey,
		IsActive:   s.IsActive,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
}

// CreateStreamRequest is the expected body for registering a stream.
type CreateStreamRequest struct {
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	SourceURL  string `json:"source_url"`
	GroupID    string `json:"group_id"`
}

// Validate checks that required fields are present.
func (r *CreateStreamRequest) Validate() error {
	if r.Name == "" {
		return ErrValidation("name is required")
	}
	if r.SourceType == "" {
		return ErrValidation("source_type is required")
	}
	validTypes := map[string]bool{
		SourceTypeRTSP:         true,
		SourceTypeRTMP:         true,
		SourceTypeWHIP:         true,
		SourceTypeDeviceCamera: true,
		SourceTypeScreenShare:  true,
	}
	if !validTypes[r.SourceType] {
		return ErrValidation("source_type must be one of: rtsp, rtmp, whip, device_camera, screen_share")
	}
	if r.SourceType == SourceTypeRTSP && r.SourceURL == "" {
		return ErrValidation("source_url is required for rtsp streams")
	}
	if r.GroupID == "" {
		return ErrValidation("group_id is required")
	}
	if _, err := uuid.Parse(r.GroupID); err != nil {
		return ErrValidation("group_id must be a valid UUID")
	}
	return nil
}

// UpdateStreamRequest is the expected body for updating a stream.
type UpdateStreamRequest struct {
	Name      *string `json:"name"`
	SourceURL *string `json:"source_url"`
}

// StreamStartResponse is returned when a stream starts ingesting.
type StreamStartResponse struct {
	Stream    StreamResponse `json:"stream"`
	IngestURL string         `json:"ingest_url,omitempty"`
	StreamKey string         `json:"stream_key,omitempty"`
	Token     string         `json:"token,omitempty"`
	URL       string         `json:"url,omitempty"`
}
