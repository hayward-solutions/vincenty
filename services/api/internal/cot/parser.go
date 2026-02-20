package cot

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Parsed CoT structures (public, used by translator and service layers).

// Event is the parsed representation of a CoT <event> element.
type Event struct {
	Version string
	UID     string
	Type    string
	Time    time.Time
	Start   time.Time
	Stale   time.Time
	How     string
	Point   Point
	Detail  Detail
	RawXML  string // original XML bytes for storage
}

// Point is the parsed <point> element.
type Point struct {
	Lat float64
	Lon float64
	HAE float64
	CE  float64
	LE  float64
}

// Detail is the parsed <detail> element with known sub-elements extracted.
type Detail struct {
	Contact *Contact
	Group   *Group
	Status  *Status
	TakV    *TakV
	Track   *Track
	RawXML  string // the raw <detail>...</detail> XML for storage
}

// Contact is the parsed <contact> element.
type Contact struct {
	Callsign string
	Endpoint string
}

// Group is the parsed <__group> element.
type Group struct {
	Name string
	Role string
}

// Status is the parsed <status> element.
type Status struct {
	Battery int
}

// TakV is the parsed <takv> element (TAK version info).
type TakV struct {
	Device   string
	OS       string
	Platform string
	Version  string
}

// Track is the parsed <track> element (speed/course).
type Track struct {
	Speed  float64
	Course float64
}

// XML mapping structs (unexported, used only for xml.Unmarshal).

type xmlEvents struct {
	XMLName xml.Name   `xml:"events"`
	Events  []xmlEvent `xml:"event"`
}

type xmlEvent struct {
	XMLName xml.Name  `xml:"event"`
	Version string    `xml:"version,attr"`
	UID     string    `xml:"uid,attr"`
	Type    string    `xml:"type,attr"`
	Time    string    `xml:"time,attr"`
	Start   string    `xml:"start,attr"`
	Stale   string    `xml:"stale,attr"`
	How     string    `xml:"how,attr"`
	Point   xmlPoint  `xml:"point"`
	Detail  xmlDetail `xml:"detail"`
}

type xmlPoint struct {
	Lat string `xml:"lat,attr"`
	Lon string `xml:"lon,attr"`
	HAE string `xml:"hae,attr"`
	CE  string `xml:"ce,attr"`
	LE  string `xml:"le,attr"`
}

type xmlDetail struct {
	Contact *xmlContact `xml:"contact"`
	Group   *xmlGroup   `xml:"__group"`
	Status  *xmlStatus  `xml:"status"`
	TakV    *xmlTakV    `xml:"takv"`
	Track   *xmlTrack   `xml:"track"`
	Inner   []byte      `xml:",innerxml"` // capture raw inner XML
}

type xmlContact struct {
	Callsign string `xml:"callsign,attr"`
	Endpoint string `xml:"endpoint,attr"`
}

type xmlGroup struct {
	Name string `xml:"name,attr"`
	Role string `xml:"role,attr"`
}

type xmlStatus struct {
	Battery string `xml:"battery,attr"`
}

type xmlTakV struct {
	Device   string `xml:"device,attr"`
	OS       string `xml:"os,attr"`
	Platform string `xml:"platform,attr"`
	Version  string `xml:"version,attr"`
}

type xmlTrack struct {
	Speed  string `xml:"speed,attr"`
	Course string `xml:"course,attr"`
}

// Parse reads CoT XML from r and returns parsed events.
// Accepts either a single <event> or multiple events wrapped in <events>.
func Parse(r io.Reader) ([]Event, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read cot xml: %w", err)
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("empty cot xml body")
	}

	var xmlEvts []xmlEvent

	// Try batch <events> wrapper first.
	if strings.HasPrefix(trimmed, "<events") {
		var batch xmlEvents
		if err := xml.Unmarshal(data, &batch); err != nil {
			return nil, fmt.Errorf("parse cot xml batch: %w", err)
		}
		xmlEvts = batch.Events
	} else {
		// Single <event>.
		var single xmlEvent
		if err := xml.Unmarshal(data, &single); err != nil {
			return nil, fmt.Errorf("parse cot xml event: %w", err)
		}
		xmlEvts = []xmlEvent{single}
	}

	if len(xmlEvts) == 0 {
		return nil, fmt.Errorf("no cot events found in xml")
	}

	events := make([]Event, 0, len(xmlEvts))
	for i, xe := range xmlEvts {
		evt, err := convertEvent(xe, trimmed, len(xmlEvts) == 1)
		if err != nil {
			return nil, fmt.Errorf("event[%d]: %w", i, err)
		}
		events = append(events, evt)
	}

	return events, nil
}

// convertEvent converts an xmlEvent to a public Event.
func convertEvent(xe xmlEvent, fullXML string, isSingle bool) (Event, error) {
	evtTime, err := parseTime(xe.Time)
	if err != nil {
		return Event{}, fmt.Errorf("invalid time %q: %w", xe.Time, err)
	}
	startTime, err := parseTime(xe.Start)
	if err != nil {
		return Event{}, fmt.Errorf("invalid start %q: %w", xe.Start, err)
	}
	staleTime, err := parseTime(xe.Stale)
	if err != nil {
		return Event{}, fmt.Errorf("invalid stale %q: %w", xe.Stale, err)
	}

	pt, err := parsePoint(xe.Point)
	if err != nil {
		return Event{}, fmt.Errorf("invalid point: %w", err)
	}

	detail := convertDetail(xe.Detail)

	// For single events, store the full original XML. For batch, re-marshal individual event.
	rawXML := fullXML
	if !isSingle {
		if b, err := xml.MarshalIndent(xe, "", "  "); err == nil {
			rawXML = string(b)
		}
	}

	return Event{
		Version: xe.Version,
		UID:     xe.UID,
		Type:    xe.Type,
		Time:    evtTime,
		Start:   startTime,
		Stale:   staleTime,
		How:     xe.How,
		Point:   pt,
		Detail:  detail,
		RawXML:  rawXML,
	}, nil
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	// CoT uses ISO 8601 / RFC 3339 timestamps.
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try with milliseconds (some TAK clients use this).
		t, err = time.Parse("2006-01-02T15:04:05.000Z", s)
		if err != nil {
			return time.Time{}, err
		}
	}
	return t, nil
}

func parsePoint(xp xmlPoint) (Point, error) {
	lat, err := strconv.ParseFloat(xp.Lat, 64)
	if err != nil {
		return Point{}, fmt.Errorf("lat: %w", err)
	}
	lon, err := strconv.ParseFloat(xp.Lon, 64)
	if err != nil {
		return Point{}, fmt.Errorf("lon: %w", err)
	}

	p := Point{Lat: lat, Lon: lon}

	if xp.HAE != "" {
		if v, err := strconv.ParseFloat(xp.HAE, 64); err == nil {
			p.HAE = v
		}
	}
	if xp.CE != "" {
		if v, err := strconv.ParseFloat(xp.CE, 64); err == nil {
			p.CE = v
		}
	}
	if xp.LE != "" {
		if v, err := strconv.ParseFloat(xp.LE, 64); err == nil {
			p.LE = v
		}
	}

	return p, nil
}

func convertDetail(xd xmlDetail) Detail {
	d := Detail{}

	if len(xd.Inner) > 0 {
		d.RawXML = "<detail>" + string(xd.Inner) + "</detail>"
	}

	if xd.Contact != nil {
		d.Contact = &Contact{
			Callsign: xd.Contact.Callsign,
			Endpoint: xd.Contact.Endpoint,
		}
	}

	if xd.Group != nil {
		d.Group = &Group{
			Name: xd.Group.Name,
			Role: xd.Group.Role,
		}
	}

	if xd.Status != nil {
		battery := 0
		if xd.Status.Battery != "" {
			if v, err := strconv.Atoi(xd.Status.Battery); err == nil {
				battery = v
			}
		}
		d.Status = &Status{Battery: battery}
	}

	if xd.TakV != nil {
		d.TakV = &TakV{
			Device:   xd.TakV.Device,
			OS:       xd.TakV.OS,
			Platform: xd.TakV.Platform,
			Version:  xd.TakV.Version,
		}
	}

	if xd.Track != nil {
		t := &Track{}
		if xd.Track.Speed != "" {
			if v, err := strconv.ParseFloat(xd.Track.Speed, 64); err == nil {
				t.Speed = v
			}
		}
		if xd.Track.Course != "" {
			if v, err := strconv.ParseFloat(xd.Track.Course, 64); err == nil {
				t.Course = v
			}
		}
		d.Track = t
	}

	return d
}
