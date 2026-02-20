package model

import (
	"time"

	"github.com/google/uuid"
)

// CoT event type category prefixes.
const (
	CotCategoryAtom    = "a-"      // Position / SA report
	CotCategoryGeoChat = "b-t-f"   // GeoChat message
	CotCategoryChatRx  = "t-x-c-t" // Chat receipt
	CotCategoryMarker  = "b-m-p-"  // Map marker / point of interest
)

// CotEvent represents a Cursor on Target event stored in the database.
type CotEvent struct {
	ID        uuid.UUID
	EventUID  string
	EventType string
	How       string
	UserID    *uuid.UUID
	DeviceID  *uuid.UUID
	Callsign  *string
	Lat       float64
	Lng       float64
	HAE       *float64
	CE        *float64
	LE        *float64
	Speed     *float64
	Course    *float64
	DetailXML *string
	RawXML    *string
	EventTime time.Time
	StartTime time.Time
	StaleTime time.Time
	CreatedAt time.Time
}

// CotEventResponse is the JSON representation returned by the API.
type CotEventResponse struct {
	ID        uuid.UUID  `json:"id"`
	EventUID  string     `json:"event_uid"`
	EventType string     `json:"event_type"`
	How       string     `json:"how"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	DeviceID  *uuid.UUID `json:"device_id,omitempty"`
	Callsign  string     `json:"callsign,omitempty"`
	Lat       float64    `json:"lat"`
	Lng       float64    `json:"lng"`
	HAE       *float64   `json:"hae,omitempty"`
	CE        *float64   `json:"ce,omitempty"`
	LE        *float64   `json:"le,omitempty"`
	Speed     *float64   `json:"speed,omitempty"`
	Course    *float64   `json:"course,omitempty"`
	DetailXML *string    `json:"detail_xml,omitempty"`
	EventTime time.Time  `json:"event_time"`
	StartTime time.Time  `json:"start_time"`
	StaleTime time.Time  `json:"stale_time"`
	CreatedAt time.Time  `json:"created_at"`
}

// ToResponse converts a CotEvent to its API response representation.
func (e *CotEvent) ToResponse() CotEventResponse {
	callsign := ""
	if e.Callsign != nil {
		callsign = *e.Callsign
	}
	return CotEventResponse{
		ID:        e.ID,
		EventUID:  e.EventUID,
		EventType: e.EventType,
		How:       e.How,
		UserID:    e.UserID,
		DeviceID:  e.DeviceID,
		Callsign:  callsign,
		Lat:       e.Lat,
		Lng:       e.Lng,
		HAE:       e.HAE,
		CE:        e.CE,
		LE:        e.LE,
		Speed:     e.Speed,
		Course:    e.Course,
		DetailXML: e.DetailXML,
		EventTime: e.EventTime,
		StartTime: e.StartTime,
		StaleTime: e.StaleTime,
		CreatedAt: e.CreatedAt,
	}
}

// CotEventFilters holds query parameters for filtering CoT events.
type CotEventFilters struct {
	EventUID  string
	EventType string // prefix match (e.g., "a-f" matches "a-f-G-U-C")
	From      *time.Time
	To        *time.Time
	Page      int
	PageSize  int
}
