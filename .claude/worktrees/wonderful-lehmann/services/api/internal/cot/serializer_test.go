package cot

import (
	"strings"
	"testing"
	"time"
)

func makeTestEvent() Event {
	return Event{
		Version: "2.0",
		UID:     "TEST-UID-1",
		Type:    "a-f-G-U-C",
		Time:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Start:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Stale:   time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC),
		How:     "m-g",
		Point: Point{
			Lat: -33.8688,
			Lon: 151.2093,
			HAE: 50.5,
			CE:  10.0,
			LE:  5.0,
		},
		Detail: Detail{
			Contact: &Contact{Callsign: "Alpha1", Endpoint: "*:-1:stcp"},
			Group:   &Group{Name: "Red Team", Role: "Lead"},
			Status:  &Status{Battery: 85},
			TakV:    &TakV{Device: "Pixel 7", OS: "Android 14", Platform: "ATAK", Version: "4.10"},
			Track:   &Track{Speed: 1.5, Course: 270.0},
		},
	}
}

func TestSerialize_SingleEvent(t *testing.T) {
	data, err := Serialize([]Event{makeTestEvent()})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	xml := string(data)
	if !strings.Contains(xml, `uid="TEST-UID-1"`) {
		t.Error("serialized XML should contain UID")
	}
	if !strings.Contains(xml, `type="a-f-G-U-C"`) {
		t.Error("serialized XML should contain type")
	}
	if !strings.Contains(xml, `callsign="Alpha1"`) {
		t.Error("serialized XML should contain callsign")
	}
	// Single event should not be wrapped in <events>
	if strings.Contains(xml, "<events>") {
		t.Error("single event should not be wrapped in <events>")
	}
}

func TestSerialize_BatchEvents(t *testing.T) {
	e1 := makeTestEvent()
	e2 := makeTestEvent()
	e2.UID = "TEST-UID-2"

	data, err := Serialize([]Event{e1, e2})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	xml := string(data)
	if !strings.Contains(xml, "<events>") {
		t.Error("batch should be wrapped in <events>")
	}
	if !strings.Contains(xml, `uid="TEST-UID-1"`) {
		t.Error("should contain first UID")
	}
	if !strings.Contains(xml, `uid="TEST-UID-2"`) {
		t.Error("should contain second UID")
	}
}

func TestSerialize_EmptyEvents(t *testing.T) {
	_, err := Serialize([]Event{})
	if err == nil {
		t.Error("Serialize() expected error for empty events, got nil")
	}
}

func TestSerialize_NilSlice(t *testing.T) {
	_, err := Serialize(nil)
	if err == nil {
		t.Error("Serialize() expected error for nil events, got nil")
	}
}

func TestSerialize_MinimalEvent(t *testing.T) {
	e := Event{
		Version: "2.0",
		UID:     "minimal",
		Type:    "a-f-G",
		Time:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Start:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Stale:   time.Date(2024, 1, 1, 0, 5, 0, 0, time.UTC),
		How:     "m-g",
		Point:   Point{Lat: 0, Lon: 0},
	}

	data, err := Serialize([]Event{e})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	xml := string(data)
	if !strings.Contains(xml, `uid="minimal"`) {
		t.Error("should contain UID")
	}
	// No detail sub-elements
	if strings.Contains(xml, "callsign") {
		t.Error("minimal event should not contain callsign")
	}
}

func TestSerialize_RoundTrip(t *testing.T) {
	original := makeTestEvent()
	original.RawXML = "" // clear raw for clean comparison

	data, err := Serialize([]Event{original})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	parsed, err := Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Parse() error on round-trip = %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected 1 event after round-trip, got %d", len(parsed))
	}

	p := parsed[0]
	if p.UID != original.UID {
		t.Errorf("round-trip UID = %q, want %q", p.UID, original.UID)
	}
	if p.Type != original.Type {
		t.Errorf("round-trip Type = %q, want %q", p.Type, original.Type)
	}
	if p.Point.Lat != original.Point.Lat {
		t.Errorf("round-trip Lat = %f, want %f", p.Point.Lat, original.Point.Lat)
	}
	if p.Detail.Contact == nil || p.Detail.Contact.Callsign != "Alpha1" {
		t.Error("round-trip lost contact detail")
	}
}
