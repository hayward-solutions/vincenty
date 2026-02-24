package cot

import (
	"strings"
	"testing"
)

const singleEventXML = `<?xml version="1.0" encoding="UTF-8"?>
<event version="2.0" uid="ANDROID-abc123" type="a-f-G-U-C" time="2024-01-15T10:30:00Z" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="m-g">
  <point lat="-33.8688" lon="151.2093" hae="50.5" ce="10.0" le="5.0"/>
  <detail>
    <contact callsign="Alpha1" endpoint="*:-1:stcp"/>
    <__group name="Team Red" role="Team Lead"/>
    <status battery="85"/>
    <takv device="Pixel 7" os="Android 14" platform="ATAK-CIV" version="4.10.0"/>
    <track speed="1.5" course="270.0"/>
  </detail>
</event>`

const batchEventsXML = `<events>
  <event version="2.0" uid="UID-1" type="a-f-G-U-C" time="2024-01-15T10:30:00Z" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="m-g">
    <point lat="-33.8688" lon="151.2093" hae="0" ce="0" le="0"/>
    <detail/>
  </event>
  <event version="2.0" uid="UID-2" type="a-f-G-U-C" time="2024-01-15T10:31:00Z" start="2024-01-15T10:31:00Z" stale="2024-01-15T10:36:00Z" how="m-g">
    <point lat="-33.8700" lon="151.2100" hae="0" ce="0" le="0"/>
    <detail/>
  </event>
</events>`

func TestParse_SingleEvent(t *testing.T) {
	events, err := Parse(strings.NewReader(singleEventXML))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Parse() returned %d events, want 1", len(events))
	}

	e := events[0]
	if e.Version != "2.0" {
		t.Errorf("Version = %q, want %q", e.Version, "2.0")
	}
	if e.UID != "ANDROID-abc123" {
		t.Errorf("UID = %q, want %q", e.UID, "ANDROID-abc123")
	}
	if e.Type != "a-f-G-U-C" {
		t.Errorf("Type = %q, want %q", e.Type, "a-f-G-U-C")
	}
	if e.How != "m-g" {
		t.Errorf("How = %q, want %q", e.How, "m-g")
	}
}

func TestParse_SingleEvent_Point(t *testing.T) {
	events, _ := Parse(strings.NewReader(singleEventXML))
	p := events[0].Point

	if p.Lat != -33.8688 {
		t.Errorf("Lat = %f, want %f", p.Lat, -33.8688)
	}
	if p.Lon != 151.2093 {
		t.Errorf("Lon = %f, want %f", p.Lon, 151.2093)
	}
	if p.HAE != 50.5 {
		t.Errorf("HAE = %f, want %f", p.HAE, 50.5)
	}
	if p.CE != 10.0 {
		t.Errorf("CE = %f, want %f", p.CE, 10.0)
	}
	if p.LE != 5.0 {
		t.Errorf("LE = %f, want %f", p.LE, 5.0)
	}
}

func TestParse_SingleEvent_DetailContact(t *testing.T) {
	events, _ := Parse(strings.NewReader(singleEventXML))
	d := events[0].Detail

	if d.Contact == nil {
		t.Fatal("Detail.Contact is nil")
	}
	if d.Contact.Callsign != "Alpha1" {
		t.Errorf("Callsign = %q, want %q", d.Contact.Callsign, "Alpha1")
	}
	if d.Contact.Endpoint != "*:-1:stcp" {
		t.Errorf("Endpoint = %q, want %q", d.Contact.Endpoint, "*:-1:stcp")
	}
}

func TestParse_SingleEvent_DetailGroup(t *testing.T) {
	events, _ := Parse(strings.NewReader(singleEventXML))
	d := events[0].Detail

	if d.Group == nil {
		t.Fatal("Detail.Group is nil")
	}
	if d.Group.Name != "Team Red" {
		t.Errorf("Group.Name = %q, want %q", d.Group.Name, "Team Red")
	}
	if d.Group.Role != "Team Lead" {
		t.Errorf("Group.Role = %q, want %q", d.Group.Role, "Team Lead")
	}
}

func TestParse_SingleEvent_DetailStatus(t *testing.T) {
	events, _ := Parse(strings.NewReader(singleEventXML))
	d := events[0].Detail

	if d.Status == nil {
		t.Fatal("Detail.Status is nil")
	}
	if d.Status.Battery != 85 {
		t.Errorf("Battery = %d, want %d", d.Status.Battery, 85)
	}
}

func TestParse_SingleEvent_DetailTakV(t *testing.T) {
	events, _ := Parse(strings.NewReader(singleEventXML))
	d := events[0].Detail

	if d.TakV == nil {
		t.Fatal("Detail.TakV is nil")
	}
	if d.TakV.Device != "Pixel 7" {
		t.Errorf("Device = %q, want %q", d.TakV.Device, "Pixel 7")
	}
	if d.TakV.OS != "Android 14" {
		t.Errorf("OS = %q, want %q", d.TakV.OS, "Android 14")
	}
	if d.TakV.Platform != "ATAK-CIV" {
		t.Errorf("Platform = %q, want %q", d.TakV.Platform, "ATAK-CIV")
	}
}

func TestParse_SingleEvent_DetailTrack(t *testing.T) {
	events, _ := Parse(strings.NewReader(singleEventXML))
	d := events[0].Detail

	if d.Track == nil {
		t.Fatal("Detail.Track is nil")
	}
	if d.Track.Speed != 1.5 {
		t.Errorf("Speed = %f, want %f", d.Track.Speed, 1.5)
	}
	if d.Track.Course != 270.0 {
		t.Errorf("Course = %f, want %f", d.Track.Course, 270.0)
	}
}

func TestParse_SingleEvent_Timestamps(t *testing.T) {
	events, _ := Parse(strings.NewReader(singleEventXML))
	e := events[0]

	if e.Time.IsZero() {
		t.Error("Time is zero")
	}
	if e.Start.IsZero() {
		t.Error("Start is zero")
	}
	if e.Stale.IsZero() {
		t.Error("Stale is zero")
	}
	if !e.Stale.After(e.Start) {
		t.Error("Stale should be after Start")
	}
}

func TestParse_SingleEvent_RawXML(t *testing.T) {
	events, _ := Parse(strings.NewReader(singleEventXML))
	if events[0].RawXML == "" {
		t.Error("RawXML should not be empty for single event")
	}
}

func TestParse_BatchEvents(t *testing.T) {
	events, err := Parse(strings.NewReader(batchEventsXML))
	if err != nil {
		t.Fatalf("Parse() batch error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("Parse() returned %d events, want 2", len(events))
	}

	if events[0].UID != "UID-1" {
		t.Errorf("events[0].UID = %q, want %q", events[0].UID, "UID-1")
	}
	if events[1].UID != "UID-2" {
		t.Errorf("events[1].UID = %q, want %q", events[1].UID, "UID-2")
	}
}

func TestParse_EmptyBody(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	if err == nil {
		t.Error("Parse() expected error for empty body, got nil")
	}
}

func TestParse_WhitespaceOnly(t *testing.T) {
	_, err := Parse(strings.NewReader("   \n\t  "))
	if err == nil {
		t.Error("Parse() expected error for whitespace-only body, got nil")
	}
}

func TestParse_InvalidXML(t *testing.T) {
	_, err := Parse(strings.NewReader("<not valid xml"))
	if err == nil {
		t.Error("Parse() expected error for invalid XML, got nil")
	}
}

func TestParse_MissingTimestamp(t *testing.T) {
	xml := `<event version="2.0" uid="test" type="a-f-G-U-C" time="" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="m-g">
  <point lat="0" lon="0" hae="0" ce="0" le="0"/>
  <detail/>
</event>`
	_, err := Parse(strings.NewReader(xml))
	if err == nil {
		t.Error("Parse() expected error for empty timestamp, got nil")
	}
}

func TestParse_MillisecondTimestamp(t *testing.T) {
	xml := `<event version="2.0" uid="test" type="a-f-G-U-C" time="2024-01-15T10:30:00.123Z" start="2024-01-15T10:30:00.123Z" stale="2024-01-15T10:35:00.123Z" how="m-g">
  <point lat="0" lon="0" hae="0" ce="0" le="0"/>
  <detail/>
</event>`
	events, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error with millisecond timestamp = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestParse_MinimalEvent(t *testing.T) {
	xml := `<event version="2.0" uid="test" type="a-f-G-U-C" time="2024-01-15T10:30:00Z" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="m-g">
  <point lat="-33.0" lon="151.0" hae="0" ce="0" le="0"/>
  <detail/>
</event>`
	events, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	e := events[0]
	if e.Detail.Contact != nil {
		t.Error("expected nil Contact for minimal event")
	}
	if e.Detail.Group != nil {
		t.Error("expected nil Group for minimal event")
	}
	if e.Detail.Status != nil {
		t.Error("expected nil Status for minimal event")
	}
	if e.Detail.TakV != nil {
		t.Error("expected nil TakV for minimal event")
	}
	if e.Detail.Track != nil {
		t.Error("expected nil Track for minimal event")
	}
}

func TestParse_EmptyBatch(t *testing.T) {
	xml := `<events></events>`
	_, err := Parse(strings.NewReader(xml))
	if err == nil {
		t.Error("Parse() expected error for empty batch, got nil")
	}
}
