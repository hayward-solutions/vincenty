package model

import (
	"time"

	"github.com/google/uuid"
)

// RecordingStatus constants.
const (
	RecordingStatusRecording  = "recording"
	RecordingStatusProcessing = "processing"
	RecordingStatusComplete   = "complete"
	RecordingStatusFailed     = "failed"
)

// Recording represents a stored media recording.
type Recording struct {
	ID            uuid.UUID  `json:"-"`
	RoomID        *uuid.UUID `json:"-"`
	StreamID      *uuid.UUID `json:"-"`
	EgressID      string     `json:"-"`
	StoragePath   *string    `json:"-"`
	FileType      string     `json:"-"`
	DurationSecs  *int       `json:"-"`
	FileSizeBytes *int64     `json:"-"`
	Status        string     `json:"-"`
	StartedAt     time.Time  `json:"-"`
	EndedAt       *time.Time `json:"-"`
}

// RecordingResponse is the JSON representation of a recording returned by the API.
type RecordingResponse struct {
	ID            uuid.UUID  `json:"id"`
	RoomID        *uuid.UUID `json:"room_id,omitempty"`
	StreamID      *uuid.UUID `json:"stream_id,omitempty"`
	FileType      string     `json:"file_type"`
	DurationSecs  *int       `json:"duration_secs,omitempty"`
	FileSizeBytes *int64     `json:"file_size_bytes,omitempty"`
	Status        string     `json:"status"`
	PlaybackURL   string     `json:"playback_url,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
}

// ToResponse converts a Recording to its API response representation.
// playbackURL should be provided externally (presigned S3 URL).
func (r *Recording) ToResponse(playbackURL string) RecordingResponse {
	return RecordingResponse{
		ID:            r.ID,
		RoomID:        r.RoomID,
		StreamID:      r.StreamID,
		FileType:      r.FileType,
		DurationSecs:  r.DurationSecs,
		FileSizeBytes: r.FileSizeBytes,
		Status:        r.Status,
		PlaybackURL:   playbackURL,
		StartedAt:     r.StartedAt,
		EndedAt:       r.EndedAt,
	}
}
