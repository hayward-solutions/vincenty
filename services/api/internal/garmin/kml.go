// Package garmin provides parsing for Garmin InReach MapShare KML feeds.
package garmin

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// TrackPoint represents a single parsed location from a MapShare KML feed.
type TrackPoint struct {
	Lat       float64
	Lng       float64
	Altitude  *float64
	Speed     *float64  // m/s (converted from km/h in extended data)
	Course    *float64  // degrees
	Timestamp time.Time // UTC
	IMEI      string    // device IMEI from extended data
	EventType string    // e.g. "Tracking", "SOS", "Message"
}

// ParseKML parses a Garmin MapShare KML feed and extracts track points.
// The feed contains Placemarks with <Point> coordinates and <ExtendedData>.
func ParseKML(r io.Reader) ([]TrackPoint, error) {
	var doc kmlDocument
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("garmin kml: decode error: %w", err)
	}

	var points []TrackPoint

	for _, folder := range doc.Document.Folders {
		for _, pm := range folder.Placemarks {
			pt, err := parsePlacemark(pm)
			if err != nil {
				continue // skip malformed placemarks
			}
			points = append(points, pt)
		}
	}

	// Also check top-level placemarks (some feeds don't use folders)
	for _, pm := range doc.Document.Placemarks {
		pt, err := parsePlacemark(pm)
		if err != nil {
			continue
		}
		points = append(points, pt)
	}

	return points, nil
}

func parsePlacemark(pm kmlPlacemark) (TrackPoint, error) {
	var pt TrackPoint

	// Parse coordinates: "lng,lat,alt"
	coords := strings.TrimSpace(pm.Point.Coordinates)
	if coords == "" {
		return pt, fmt.Errorf("empty coordinates")
	}

	parts := strings.Split(coords, ",")
	if len(parts) < 2 {
		return pt, fmt.Errorf("invalid coordinates: %s", coords)
	}

	lng, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return pt, fmt.Errorf("invalid longitude: %w", err)
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return pt, fmt.Errorf("invalid latitude: %w", err)
	}
	pt.Lat = lat
	pt.Lng = lng

	if len(parts) >= 3 {
		if alt, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64); err == nil && alt != 0 {
			pt.Altitude = &alt
		}
	}

	// Parse extended data
	extData := make(map[string]string)
	for _, d := range pm.ExtendedData.Data {
		extData[d.Name] = strings.TrimSpace(d.Value)
	}

	// Timestamp: try "Time UTC" field first, fall back to Placemark TimeStamp
	if v, ok := extData["Time UTC"]; ok {
		if t, err := time.Parse("1/2/2006 3:04:05 PM", v); err == nil {
			pt.Timestamp = t.UTC()
		}
	}
	if pt.Timestamp.IsZero() {
		if ts := strings.TrimSpace(pm.TimeStamp.When); ts != "" {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				pt.Timestamp = t.UTC()
			}
		}
	}
	if pt.Timestamp.IsZero() {
		return pt, fmt.Errorf("no timestamp found")
	}

	// Velocity (km/h → m/s)
	if v, ok := extData["Velocity"]; ok {
		v = strings.TrimSuffix(v, " km/h")
		if speed, err := strconv.ParseFloat(v, 64); err == nil {
			ms := speed / 3.6
			pt.Speed = &ms
		}
	}

	// Course
	if v, ok := extData["Course"]; ok {
		v = strings.TrimSuffix(v, "°")
		v = strings.TrimSuffix(v, " °")
		v = strings.TrimSpace(v)
		if course, err := strconv.ParseFloat(v, 64); err == nil {
			pt.Course = &course
		}
	}

	// IMEI
	if v, ok := extData["IMEI"]; ok {
		pt.IMEI = v
	}
	if v, ok := extData["Device IMEI"]; ok && pt.IMEI == "" {
		pt.IMEI = v
	}

	// Event type
	if v, ok := extData["Event"]; ok {
		pt.EventType = v
	}
	if v, ok := extData["Type"]; ok && pt.EventType == "" {
		pt.EventType = v
	}

	return pt, nil
}

// --------------------------------------------------------------------------
// KML XML structures
// --------------------------------------------------------------------------

type kmlDocument struct {
	XMLName  xml.Name    `xml:"kml"`
	Document kmlDocInner `xml:"Document"`
}

type kmlDocInner struct {
	Folders    []kmlFolder    `xml:"Folder"`
	Placemarks []kmlPlacemark `xml:"Placemark"`
}

type kmlFolder struct {
	Name       string         `xml:"name"`
	Placemarks []kmlPlacemark `xml:"Placemark"`
}

type kmlPlacemark struct {
	Name         string          `xml:"name"`
	TimeStamp    kmlTimeStamp    `xml:"TimeStamp"`
	Point        kmlPoint        `xml:"Point"`
	ExtendedData kmlExtendedData `xml:"ExtendedData"`
}

type kmlTimeStamp struct {
	When string `xml:"when"`
}

type kmlPoint struct {
	Coordinates string `xml:"coordinates"`
}

type kmlExtendedData struct {
	Data []kmlData `xml:"Data"`
}

type kmlData struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value"`
}
