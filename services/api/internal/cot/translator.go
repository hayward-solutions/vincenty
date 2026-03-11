package cot

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/model"
)

// Category represents the classified type of a CoT event.
type Category string

const (
	CategoryPosition Category = "position" // a-* (atom) events
	CategoryGeoChat  Category = "geochat"  // b-t-f GeoChat messages
	CategoryChatRx   Category = "chat_rx"  // t-x-c-t chat receipts
	CategoryMarker   Category = "marker"   // b-m-p-* map markers
	CategoryGeneric  Category = "generic"  // everything else
)

// Classify determines the category of a CoT event based on its type code.
func Classify(eventType string) Category {
	switch {
	case strings.HasPrefix(eventType, model.CotCategoryAtom):
		return CategoryPosition
	case strings.HasPrefix(eventType, model.CotCategoryGeoChat):
		return CategoryGeoChat
	case strings.HasPrefix(eventType, model.CotCategoryChatRx):
		return CategoryChatRx
	case strings.HasPrefix(eventType, model.CotCategoryMarker):
		return CategoryMarker
	default:
		return CategoryGeneric
	}
}

// ToCotEvent converts a parsed Event to a model.CotEvent for database storage.
// userID and deviceID are resolved externally via device_uid lookup.
func ToCotEvent(e Event, userID *uuid.UUID, deviceID *uuid.UUID) model.CotEvent {
	evt := model.CotEvent{
		EventUID:  e.UID,
		EventType: e.Type,
		How:       e.How,
		UserID:    userID,
		DeviceID:  deviceID,
		Lat:       e.Point.Lat,
		Lng:       e.Point.Lon,
		EventTime: e.Time,
		StartTime: e.Start,
		StaleTime: e.Stale,
		RawXML:    strPtr(e.RawXML),
		CreatedAt: time.Now(),
	}

	// HAE/CE/LE: only store if not the sentinel value 9999999
	if e.Point.HAE != 0 && e.Point.HAE != 9999999 {
		evt.HAE = &e.Point.HAE
	}
	if e.Point.CE != 0 && e.Point.CE != 9999999 {
		evt.CE = &e.Point.CE
	}
	if e.Point.LE != 0 && e.Point.LE != 9999999 {
		evt.LE = &e.Point.LE
	}

	// Extract speed/course from <track> detail
	if e.Detail.Track != nil {
		if e.Detail.Track.Speed != 0 {
			evt.Speed = &e.Detail.Track.Speed
		}
		if e.Detail.Track.Course != 0 {
			evt.Course = &e.Detail.Track.Course
		}
	}

	// Callsign from <contact>
	if e.Detail.Contact != nil && e.Detail.Contact.Callsign != "" {
		evt.Callsign = &e.Detail.Contact.Callsign
	}

	// Store detail XML (without raw_xml to avoid duplication)
	if e.Detail.RawXML != "" {
		evt.DetailXML = &e.Detail.RawXML
	}

	return evt
}

// LocationBridgeData holds the data needed to call LocationService.Update
// for position events (a-* atoms).
type LocationBridgeData struct {
	UserID   uuid.UUID
	DeviceID uuid.UUID
	Lat      float64
	Lng      float64
	Altitude *float64
	Heading  *float64
	Speed    *float64
	Accuracy *float64
}

// ToLocationBridge extracts location bridge data from a CoT position event.
// Returns nil if the event has no resolved user/device (cannot bridge).
func ToLocationBridge(e Event, userID *uuid.UUID, deviceID *uuid.UUID) *LocationBridgeData {
	if userID == nil || deviceID == nil {
		return nil
	}

	data := &LocationBridgeData{
		UserID:   *userID,
		DeviceID: *deviceID,
		Lat:      e.Point.Lat,
		Lng:      e.Point.Lon,
	}

	// Map HAE → altitude (skip sentinel)
	if e.Point.HAE != 0 && e.Point.HAE != 9999999 {
		data.Altitude = &e.Point.HAE
	}

	// Map CE → accuracy (circular error → horizontal accuracy)
	if e.Point.CE != 0 && e.Point.CE != 9999999 {
		data.Accuracy = &e.Point.CE
	}

	if e.Detail.Track != nil {
		if e.Detail.Track.Speed != 0 {
			data.Speed = &e.Detail.Track.Speed
		}
		if e.Detail.Track.Course != 0 {
			data.Heading = &e.Detail.Track.Course
		}
	}

	return data
}

// GeoChatBridgeData holds the data needed to bridge a GeoChat CoT event
// into the internal messaging system.
type GeoChatBridgeData struct {
	SenderUserID uuid.UUID
	DeviceID     uuid.UUID
	Content      string
	Callsign     string
	Lat          float64
	Lng          float64
}

// ToGeoChatBridge extracts message bridge data from a GeoChat CoT event.
// Returns nil if the event has no resolved user/device.
func ToGeoChatBridge(e Event, userID *uuid.UUID, deviceID *uuid.UUID) *GeoChatBridgeData {
	if userID == nil || deviceID == nil {
		return nil
	}

	callsign := ""
	if e.Detail.Contact != nil {
		callsign = e.Detail.Contact.Callsign
	}

	// GeoChat content is typically in the remarks element or the UID itself.
	// For now, use callsign + type as content placeholder.
	// TODO: Extract <remarks> from detail XML in a future iteration.
	content := "[CoT GeoChat from " + callsign + "]"

	return &GeoChatBridgeData{
		SenderUserID: *userID,
		DeviceID:     *deviceID,
		Content:      content,
		Callsign:     callsign,
		Lat:          e.Point.Lat,
		Lng:          e.Point.Lon,
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
