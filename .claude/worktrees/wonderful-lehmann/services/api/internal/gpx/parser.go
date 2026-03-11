package gpx

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
)

// GPX XML structures (subset needed for parsing waypoints, tracks, routes).

type gpxFile struct {
	XMLName   xml.Name   `xml:"gpx"`
	Waypoints []waypoint `xml:"wpt"`
	Tracks    []track    `xml:"trk"`
	Routes    []route    `xml:"rte"`
}

type waypoint struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Ele  float64 `xml:"ele"`
	Name string  `xml:"name"`
	Desc string  `xml:"desc"`
	Time string  `xml:"time"`
}

type track struct {
	Name     string    `xml:"name"`
	Segments []segment `xml:"trkseg"`
}

type segment struct {
	Points []waypoint `xml:"trkpt"`
}

type route struct {
	Name   string     `xml:"name"`
	Points []waypoint `xml:"rtept"`
}

// GeoJSON structures.

type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	Type       string                 `json:"type"`
	Geometry   geometry               `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

type geometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

// Parse reads a GPX file and returns a GeoJSON FeatureCollection as json.RawMessage.
func Parse(r io.Reader) (json.RawMessage, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read gpx: %w", err)
	}

	var gpx gpxFile
	if err := xml.Unmarshal(data, &gpx); err != nil {
		return nil, fmt.Errorf("parse gpx xml: %w", err)
	}

	fc := featureCollection{
		Type:     "FeatureCollection",
		Features: make([]feature, 0),
	}

	// Waypoints → Point features
	for _, wpt := range gpx.Waypoints {
		props := map[string]interface{}{
			"type": "waypoint",
		}
		if wpt.Name != "" {
			props["name"] = wpt.Name
		}
		if wpt.Desc != "" {
			props["description"] = wpt.Desc
		}
		if wpt.Time != "" {
			props["time"] = wpt.Time
		}

		coords := []float64{wpt.Lon, wpt.Lat}
		if wpt.Ele != 0 {
			coords = append(coords, wpt.Ele)
		}

		fc.Features = append(fc.Features, feature{
			Type: "Feature",
			Geometry: geometry{
				Type:        "Point",
				Coordinates: coords,
			},
			Properties: props,
		})
	}

	// Tracks → LineString features (one per segment)
	for _, trk := range gpx.Tracks {
		for segIdx, seg := range trk.Segments {
			if len(seg.Points) < 2 {
				continue
			}

			coords := make([][]float64, 0, len(seg.Points))
			for _, pt := range seg.Points {
				c := []float64{pt.Lon, pt.Lat}
				if pt.Ele != 0 {
					c = append(c, pt.Ele)
				}
				coords = append(coords, c)
			}

			props := map[string]interface{}{
				"type": "track",
			}
			if trk.Name != "" {
				props["name"] = trk.Name
			}
			if len(trk.Segments) > 1 {
				props["segment"] = segIdx
			}

			fc.Features = append(fc.Features, feature{
				Type: "Feature",
				Geometry: geometry{
					Type:        "LineString",
					Coordinates: coords,
				},
				Properties: props,
			})
		}
	}

	// Routes → LineString features
	for _, rte := range gpx.Routes {
		if len(rte.Points) < 2 {
			continue
		}

		coords := make([][]float64, 0, len(rte.Points))
		for _, pt := range rte.Points {
			c := []float64{pt.Lon, pt.Lat}
			if pt.Ele != 0 {
				c = append(c, pt.Ele)
			}
			coords = append(coords, c)
		}

		props := map[string]interface{}{
			"type": "route",
		}
		if rte.Name != "" {
			props["name"] = rte.Name
		}

		fc.Features = append(fc.Features, feature{
			Type: "Feature",
			Geometry: geometry{
				Type:        "LineString",
				Coordinates: coords,
			},
			Properties: props,
		})
	}

	raw, err := json.Marshal(fc)
	if err != nil {
		return nil, fmt.Errorf("marshal geojson: %w", err)
	}

	return json.RawMessage(raw), nil
}
