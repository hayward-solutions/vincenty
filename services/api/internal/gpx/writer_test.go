package gpx

import (
	"strings"
	"testing"
	"time"
)

func TestGenerate_BasicTrack(t *testing.T) {
	elev := 100.5
	points := []TrackPoint{
		{Lat: -33.8688, Lng: 151.2093, Elevation: &elev, Time: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)},
		{Lat: -33.8700, Lng: 151.2100, Elevation: nil, Time: time.Date(2024, 1, 15, 10, 1, 0, 0, time.UTC)},
	}

	data, err := Generate("Test Track", points)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	xml := string(data)

	if !strings.Contains(xml, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("should contain XML declaration")
	}
	if !strings.Contains(xml, `version="1.1"`) {
		t.Error("should contain GPX version 1.1")
	}
	if !strings.Contains(xml, `creator="SitAware"`) {
		t.Error("should contain creator")
	}
	if !strings.Contains(xml, `<name>Test Track</name>`) {
		t.Error("should contain track name")
	}
	if !strings.Contains(xml, `lat="-33.8688"`) {
		t.Error("should contain latitude")
	}
	if !strings.Contains(xml, `lon="151.2093"`) {
		t.Error("should contain longitude")
	}
	if !strings.Contains(xml, `<ele>100.5</ele>`) {
		t.Error("should contain elevation")
	}
}

func TestGenerate_EmptyPoints(t *testing.T) {
	_, err := Generate("Empty", []TrackPoint{})
	if err == nil {
		t.Error("Generate() expected error for empty points, got nil")
	}
}

func TestGenerate_NilPoints(t *testing.T) {
	_, err := Generate("Nil", nil)
	if err == nil {
		t.Error("Generate() expected error for nil points, got nil")
	}
}

func TestGenerate_SinglePoint(t *testing.T) {
	points := []TrackPoint{
		{Lat: 0, Lng: 0, Time: time.Now()},
	}

	data, err := Generate("Single", points)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	xml := string(data)
	if !strings.Contains(xml, "<trkseg>") {
		t.Error("should contain track segment")
	}
	if !strings.Contains(xml, "<trkpt") {
		t.Error("should contain track point")
	}
}

func TestGenerate_NoElevation(t *testing.T) {
	points := []TrackPoint{
		{Lat: 10.0, Lng: 20.0, Elevation: nil, Time: time.Now()},
	}

	data, err := Generate("No Elev", points)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	xml := string(data)
	if strings.Contains(xml, "<ele>") {
		t.Error("should not contain <ele> when elevation is nil")
	}
}

func TestGenerate_TimeFormat(t *testing.T) {
	points := []TrackPoint{
		{Lat: 0, Lng: 0, Time: time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)},
	}

	data, _ := Generate("Time Test", points)
	xml := string(data)

	if !strings.Contains(xml, "2024-06-15T14:30:45Z") {
		t.Error("time should be formatted as RFC3339 UTC")
	}
}

func TestGenerate_ValidGPXStructure(t *testing.T) {
	elev := 50.0
	points := []TrackPoint{
		{Lat: 1.0, Lng: 2.0, Elevation: &elev, Time: time.Now()},
		{Lat: 3.0, Lng: 4.0, Elevation: &elev, Time: time.Now()},
	}

	data, _ := Generate("Structure Test", points)
	xml := string(data)

	// Check proper nesting
	if !strings.Contains(xml, "<gpx") {
		t.Error("should have <gpx> root element")
	}
	if !strings.Contains(xml, "<trk>") {
		t.Error("should have <trk> element")
	}
	if !strings.Contains(xml, "<trkseg>") {
		t.Error("should have <trkseg> element")
	}
	if strings.Count(xml, "<trkpt") != 2 {
		t.Errorf("should have 2 <trkpt> elements, got %d", strings.Count(xml, "<trkpt"))
	}
}

func TestGenerate_XMLNS(t *testing.T) {
	points := []TrackPoint{{Lat: 0, Lng: 0, Time: time.Now()}}
	data, _ := Generate("NS Test", points)
	xml := string(data)

	if !strings.Contains(xml, "http://www.topografix.com/GPX/1/1") {
		t.Error("should contain GPX namespace")
	}
}
