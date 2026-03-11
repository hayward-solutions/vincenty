package gpx

import (
	"encoding/xml"
	"fmt"
	"time"
)

// TrackPoint represents a single point in a GPS track for export.
type TrackPoint struct {
	Lat       float64
	Lng       float64
	Elevation *float64
	Time      time.Time
}

// gpxRoot is the top-level GPX XML element.
type gpxRoot struct {
	XMLName xml.Name `xml:"gpx"`
	Version string   `xml:"version,attr"`
	Creator string   `xml:"creator,attr"`
	XMLNS   string   `xml:"xmlns,attr"`
	Tracks  []gpxTrk `xml:"trk"`
}

// gpxTrk is a GPX track element.
type gpxTrk struct {
	Name     string      `xml:"name"`
	Segments []gpxTrkSeg `xml:"trkseg"`
}

// gpxTrkSeg is a GPX track segment.
type gpxTrkSeg struct {
	Points []gpxTrkPt `xml:"trkpt"`
}

// gpxTrkPt is a single track point.
type gpxTrkPt struct {
	Lat  float64  `xml:"lat,attr"`
	Lon  float64  `xml:"lon,attr"`
	Ele  *float64 `xml:"ele,omitempty"`
	Time string   `xml:"time,omitempty"`
}

// Generate creates a GPX XML document from a slice of track points.
// The points are written as a single track with one segment.
func Generate(name string, points []TrackPoint) ([]byte, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("no points to export")
	}

	seg := gpxTrkSeg{
		Points: make([]gpxTrkPt, len(points)),
	}
	for i, p := range points {
		seg.Points[i] = gpxTrkPt{
			Lat:  p.Lat,
			Lon:  p.Lng,
			Ele:  p.Elevation,
			Time: p.Time.UTC().Format(time.RFC3339),
		}
	}

	root := gpxRoot{
		Version: "1.1",
		Creator: "Vincenty",
		XMLNS:   "http://www.topografix.com/GPX/1/1",
		Tracks: []gpxTrk{
			{
				Name:     name,
				Segments: []gpxTrkSeg{seg},
			},
		},
	}

	data, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal gpx: %w", err)
	}

	return append([]byte(xml.Header), data...), nil
}
