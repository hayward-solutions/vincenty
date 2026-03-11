package cot

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"time"
)

// Serialize converts parsed Event(s) to CoT XML bytes.
// Produces a single <event> for one event, or <events> wrapper for multiple.
func Serialize(events []Event) ([]byte, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("no events to serialize")
	}

	if len(events) == 1 {
		xe := toXMLEvent(events[0])
		return xml.MarshalIndent(xe, "", "  ")
	}

	batch := xmlEvents{
		Events: make([]xmlEvent, len(events)),
	}
	for i, e := range events {
		batch.Events[i] = toXMLEvent(e)
	}
	return xml.MarshalIndent(batch, "", "  ")
}

// toXMLEvent converts a public Event back to an xmlEvent for marshalling.
func toXMLEvent(e Event) xmlEvent {
	xe := xmlEvent{
		Version: e.Version,
		UID:     e.UID,
		Type:    e.Type,
		Time:    formatTime(e.Time),
		Start:   formatTime(e.Start),
		Stale:   formatTime(e.Stale),
		How:     e.How,
		Point: xmlPoint{
			Lat: formatFloat(e.Point.Lat),
			Lon: formatFloat(e.Point.Lon),
			HAE: formatFloat(e.Point.HAE),
			CE:  formatFloat(e.Point.CE),
			LE:  formatFloat(e.Point.LE),
		},
	}

	xe.Detail = toXMLDetail(e.Detail)
	return xe
}

func toXMLDetail(d Detail) xmlDetail {
	xd := xmlDetail{}

	if d.Contact != nil {
		xd.Contact = &xmlContact{
			Callsign: d.Contact.Callsign,
			Endpoint: d.Contact.Endpoint,
		}
	}

	if d.Group != nil {
		xd.Group = &xmlGroup{
			Name: d.Group.Name,
			Role: d.Group.Role,
		}
	}

	if d.Status != nil {
		xd.Status = &xmlStatus{
			Battery: strconv.Itoa(d.Status.Battery),
		}
	}

	if d.TakV != nil {
		xd.TakV = &xmlTakV{
			Device:   d.TakV.Device,
			OS:       d.TakV.OS,
			Platform: d.TakV.Platform,
			Version:  d.TakV.Version,
		}
	}

	if d.Track != nil {
		xd.Track = &xmlTrack{
			Speed:  formatFloat(d.Track.Speed),
			Course: formatFloat(d.Track.Course),
		}
	}

	return xd
}

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
