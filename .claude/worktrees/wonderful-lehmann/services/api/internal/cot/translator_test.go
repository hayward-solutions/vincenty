package cot

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		want      Category
	}{
		{"position atom", "a-f-G-U-C", CategoryPosition},
		{"position hostile", "a-h-G", CategoryPosition},
		{"geochat", "b-t-f", CategoryGeoChat},
		{"geochat subtype", "b-t-f-d", CategoryGeoChat},
		{"chat receipt", "t-x-c-t", CategoryChatRx},
		{"chat receipt sub", "t-x-c-t-r", CategoryChatRx},
		{"marker waypoint", "b-m-p-w", CategoryMarker},
		{"marker spot", "b-m-p-s-p-i", CategoryMarker},
		{"generic unknown", "u-d-f-m", CategoryGeneric},
		{"empty string", "", CategoryGeneric},
		{"b-other (not geochat)", "b-a-o", CategoryGeneric},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.eventType)
			if got != tt.want {
				t.Errorf("Classify(%q) = %q, want %q", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestToCotEvent_BasicFields(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	userID := uuid.New()
	deviceID := uuid.New()

	e := Event{
		UID:   "TEST-UID",
		Type:  "a-f-G-U-C",
		How:   "m-g",
		Time:  now,
		Start: now,
		Stale: now.Add(5 * time.Minute),
		Point: Point{Lat: -33.8688, Lon: 151.2093},
	}

	evt := ToCotEvent(e, &userID, &deviceID)

	if evt.EventUID != "TEST-UID" {
		t.Errorf("EventUID = %q, want %q", evt.EventUID, "TEST-UID")
	}
	if evt.EventType != "a-f-G-U-C" {
		t.Errorf("EventType = %q, want %q", evt.EventType, "a-f-G-U-C")
	}
	if evt.How != "m-g" {
		t.Errorf("How = %q, want %q", evt.How, "m-g")
	}
	if evt.UserID == nil || *evt.UserID != userID {
		t.Errorf("UserID = %v, want %v", evt.UserID, userID)
	}
	if evt.DeviceID == nil || *evt.DeviceID != deviceID {
		t.Errorf("DeviceID = %v, want %v", evt.DeviceID, deviceID)
	}
	if evt.Lat != -33.8688 {
		t.Errorf("Lat = %v, want %v", evt.Lat, -33.8688)
	}
	if evt.Lng != 151.2093 {
		t.Errorf("Lng = %v, want %v", evt.Lng, 151.2093)
	}
}

func TestToCotEvent_NilUserAndDevice(t *testing.T) {
	e := Event{
		UID:   "ANON",
		Type:  "a-f-G",
		How:   "h-e",
		Time:  time.Now(),
		Start: time.Now(),
		Stale: time.Now(),
		Point: Point{Lat: 0, Lon: 0},
	}

	evt := ToCotEvent(e, nil, nil)

	if evt.UserID != nil {
		t.Errorf("UserID = %v, want nil", evt.UserID)
	}
	if evt.DeviceID != nil {
		t.Errorf("DeviceID = %v, want nil", evt.DeviceID)
	}
}

func TestToCotEvent_SentinelValuesSkipped(t *testing.T) {
	e := Event{
		UID:   "SENTINEL",
		Type:  "a-f-G",
		How:   "m-g",
		Time:  time.Now(),
		Start: time.Now(),
		Stale: time.Now(),
		Point: Point{
			Lat: 10,
			Lon: 20,
			HAE: 9999999,
			CE:  9999999,
			LE:  9999999,
		},
	}

	evt := ToCotEvent(e, nil, nil)

	if evt.HAE != nil {
		t.Errorf("HAE should be nil for sentinel value, got %v", *evt.HAE)
	}
	if evt.CE != nil {
		t.Errorf("CE should be nil for sentinel value, got %v", *evt.CE)
	}
	if evt.LE != nil {
		t.Errorf("LE should be nil for sentinel value, got %v", *evt.LE)
	}
}

func TestToCotEvent_HAECELEStored(t *testing.T) {
	e := Event{
		UID:   "HAE",
		Type:  "a-f-G",
		How:   "m-g",
		Time:  time.Now(),
		Start: time.Now(),
		Stale: time.Now(),
		Point: Point{
			Lat: 10,
			Lon: 20,
			HAE: 150.5,
			CE:  10.0,
			LE:  5.0,
		},
	}

	evt := ToCotEvent(e, nil, nil)

	if evt.HAE == nil || *evt.HAE != 150.5 {
		t.Errorf("HAE = %v, want 150.5", evt.HAE)
	}
	if evt.CE == nil || *evt.CE != 10.0 {
		t.Errorf("CE = %v, want 10.0", evt.CE)
	}
	if evt.LE == nil || *evt.LE != 5.0 {
		t.Errorf("LE = %v, want 5.0", evt.LE)
	}
}

func TestToCotEvent_TrackAndContact(t *testing.T) {
	e := Event{
		UID:   "TRACK",
		Type:  "a-f-G",
		How:   "m-g",
		Time:  time.Now(),
		Start: time.Now(),
		Stale: time.Now(),
		Point: Point{Lat: 0, Lon: 0},
		Detail: Detail{
			Track:   &Track{Speed: 5.5, Course: 180.0},
			Contact: &Contact{Callsign: "Alpha1"},
			RawXML:  "<detail><track speed=\"5.5\"/></detail>",
		},
	}

	evt := ToCotEvent(e, nil, nil)

	if evt.Speed == nil || *evt.Speed != 5.5 {
		t.Errorf("Speed = %v, want 5.5", evt.Speed)
	}
	if evt.Course == nil || *evt.Course != 180.0 {
		t.Errorf("Course = %v, want 180.0", evt.Course)
	}
	if evt.Callsign == nil || *evt.Callsign != "Alpha1" {
		t.Errorf("Callsign = %v, want Alpha1", evt.Callsign)
	}
	if evt.DetailXML == nil || *evt.DetailXML != "<detail><track speed=\"5.5\"/></detail>" {
		t.Errorf("DetailXML = %v, want detail xml", evt.DetailXML)
	}
}

func TestToCotEvent_ZeroSpeedAndCourseNotStored(t *testing.T) {
	e := Event{
		UID:   "ZERO",
		Type:  "a-f-G",
		How:   "m-g",
		Time:  time.Now(),
		Start: time.Now(),
		Stale: time.Now(),
		Point: Point{Lat: 0, Lon: 0},
		Detail: Detail{
			Track: &Track{Speed: 0, Course: 0},
		},
	}

	evt := ToCotEvent(e, nil, nil)

	if evt.Speed != nil {
		t.Errorf("Speed should be nil for zero, got %v", *evt.Speed)
	}
	if evt.Course != nil {
		t.Errorf("Course should be nil for zero, got %v", *evt.Course)
	}
}

func TestToCotEvent_RawXML(t *testing.T) {
	raw := "<event>full xml</event>"
	e := Event{
		UID:    "RAW",
		Type:   "a-f-G",
		How:    "m-g",
		Time:   time.Now(),
		Start:  time.Now(),
		Stale:  time.Now(),
		Point:  Point{Lat: 0, Lon: 0},
		RawXML: raw,
	}

	evt := ToCotEvent(e, nil, nil)

	if evt.RawXML == nil || *evt.RawXML != raw {
		t.Errorf("RawXML = %v, want %q", evt.RawXML, raw)
	}
}

func TestToCotEvent_EmptyRawXML(t *testing.T) {
	e := Event{
		UID:   "EMPTY-RAW",
		Type:  "a-f-G",
		How:   "m-g",
		Time:  time.Now(),
		Start: time.Now(),
		Stale: time.Now(),
		Point: Point{Lat: 0, Lon: 0},
	}

	evt := ToCotEvent(e, nil, nil)

	if evt.RawXML != nil {
		t.Errorf("RawXML should be nil for empty, got %v", *evt.RawXML)
	}
}

// ---------------------------------------------------------------------------
// ToLocationBridge
// ---------------------------------------------------------------------------

func TestToLocationBridge_NilUserOrDevice(t *testing.T) {
	e := Event{Point: Point{Lat: 10, Lon: 20}}
	userID := uuid.New()
	deviceID := uuid.New()

	if ToLocationBridge(e, nil, nil) != nil {
		t.Error("expected nil when both user and device are nil")
	}
	if ToLocationBridge(e, &userID, nil) != nil {
		t.Error("expected nil when device is nil")
	}
	if ToLocationBridge(e, nil, &deviceID) != nil {
		t.Error("expected nil when user is nil")
	}
}

func TestToLocationBridge_BasicFields(t *testing.T) {
	userID := uuid.New()
	deviceID := uuid.New()
	e := Event{
		Point: Point{Lat: -33.86, Lon: 151.20, HAE: 50.0, CE: 10.0},
		Detail: Detail{
			Track: &Track{Speed: 3.0, Course: 90.0},
		},
	}

	data := ToLocationBridge(e, &userID, &deviceID)

	if data == nil {
		t.Fatal("expected non-nil LocationBridgeData")
	}
	if data.UserID != userID {
		t.Errorf("UserID = %v, want %v", data.UserID, userID)
	}
	if data.DeviceID != deviceID {
		t.Errorf("DeviceID = %v, want %v", data.DeviceID, deviceID)
	}
	if data.Lat != -33.86 {
		t.Errorf("Lat = %v, want %v", data.Lat, -33.86)
	}
	if data.Lng != 151.20 {
		t.Errorf("Lng = %v, want %v", data.Lng, 151.20)
	}
	if data.Altitude == nil || *data.Altitude != 50.0 {
		t.Errorf("Altitude = %v, want 50.0", data.Altitude)
	}
	if data.Accuracy == nil || *data.Accuracy != 10.0 {
		t.Errorf("Accuracy = %v, want 10.0", data.Accuracy)
	}
	if data.Speed == nil || *data.Speed != 3.0 {
		t.Errorf("Speed = %v, want 3.0", data.Speed)
	}
	if data.Heading == nil || *data.Heading != 90.0 {
		t.Errorf("Heading = %v, want 90.0", data.Heading)
	}
}

func TestToLocationBridge_SentinelsSkipped(t *testing.T) {
	userID := uuid.New()
	deviceID := uuid.New()
	e := Event{
		Point: Point{Lat: 0, Lon: 0, HAE: 9999999, CE: 9999999},
	}

	data := ToLocationBridge(e, &userID, &deviceID)

	if data.Altitude != nil {
		t.Errorf("Altitude should be nil for sentinel, got %v", *data.Altitude)
	}
	if data.Accuracy != nil {
		t.Errorf("Accuracy should be nil for sentinel, got %v", *data.Accuracy)
	}
}

// ---------------------------------------------------------------------------
// ToGeoChatBridge
// ---------------------------------------------------------------------------

func TestToGeoChatBridge_NilUserOrDevice(t *testing.T) {
	e := Event{Point: Point{Lat: 10, Lon: 20}}
	userID := uuid.New()
	deviceID := uuid.New()

	if ToGeoChatBridge(e, nil, nil) != nil {
		t.Error("expected nil when both are nil")
	}
	if ToGeoChatBridge(e, &userID, nil) != nil {
		t.Error("expected nil when device is nil")
	}
	if ToGeoChatBridge(e, nil, &deviceID) != nil {
		t.Error("expected nil when user is nil")
	}
}

func TestToGeoChatBridge_WithCallsign(t *testing.T) {
	userID := uuid.New()
	deviceID := uuid.New()
	e := Event{
		Point: Point{Lat: -33.86, Lon: 151.20},
		Detail: Detail{
			Contact: &Contact{Callsign: "Bravo2"},
		},
	}

	data := ToGeoChatBridge(e, &userID, &deviceID)

	if data == nil {
		t.Fatal("expected non-nil GeoChatBridgeData")
	}
	if data.SenderUserID != userID {
		t.Errorf("SenderUserID = %v, want %v", data.SenderUserID, userID)
	}
	if data.Callsign != "Bravo2" {
		t.Errorf("Callsign = %q, want %q", data.Callsign, "Bravo2")
	}
	if data.Content != "[CoT GeoChat from Bravo2]" {
		t.Errorf("Content = %q, want %q", data.Content, "[CoT GeoChat from Bravo2]")
	}
}

func TestToGeoChatBridge_NoCallsign(t *testing.T) {
	userID := uuid.New()
	deviceID := uuid.New()
	e := Event{
		Point: Point{Lat: 0, Lon: 0},
	}

	data := ToGeoChatBridge(e, &userID, &deviceID)

	if data.Callsign != "" {
		t.Errorf("Callsign = %q, want empty", data.Callsign)
	}
	if data.Content != "[CoT GeoChat from ]" {
		t.Errorf("Content = %q, want %q", data.Content, "[CoT GeoChat from ]")
	}
}

// ---------------------------------------------------------------------------
// strPtr helper
// ---------------------------------------------------------------------------

func TestStrPtr(t *testing.T) {
	if strPtr("") != nil {
		t.Error("strPtr(\"\") should return nil")
	}
	p := strPtr("hello")
	if p == nil || *p != "hello" {
		t.Errorf("strPtr(\"hello\") = %v, want \"hello\"", p)
	}
}
