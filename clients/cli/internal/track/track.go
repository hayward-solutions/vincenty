// Package track provides GPX and GeoJSON parsers that extract ordered
// track points for streaming to the Vincenty API.
package track

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Point represents a single position in a track.
type Point struct {
	Lat      float64
	Lng      float64
	Altitude *float64
	Time     *time.Time // nil when the source has no timestamps
}

// Heading returns the initial bearing in degrees from p to next.
func Heading(p, next Point) *float64 {
	lat1 := p.Lat * math.Pi / 180
	lat2 := next.Lat * math.Pi / 180
	dLng := (next.Lng - p.Lng) * math.Pi / 180

	y := math.Sin(dLng) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLng)
	bearing := math.Atan2(y, x) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return &bearing
}

// Speed returns meters/second between two timed points. Returns nil if
// either point lacks a timestamp or the times are equal.
func Speed(p, next Point) *float64 {
	if p.Time == nil || next.Time == nil {
		return nil
	}
	dt := next.Time.Sub(*p.Time).Seconds()
	if dt <= 0 {
		return nil
	}
	d := haversine(p.Lat, p.Lng, next.Lat, next.Lng)
	s := d / dt
	return &s
}

// haversine returns the great-circle distance in meters between two points.
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000 // Earth radius in meters
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// Load reads a track file and returns an ordered slice of points.
// Format is detected from the file extension (.gpx or .geojson/.json).
func Load(path string) ([]Point, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open track file: %w", err)
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".gpx":
		return parseGPX(f)
	case ".geojson", ".json":
		return parseGeoJSON(f)
	default:
		return nil, fmt.Errorf("unsupported file extension %q (expected .gpx, .geojson, or .json)", ext)
	}
}

// ---------------------------------------------------------------------------
// GPX
// ---------------------------------------------------------------------------

type gpxFile struct {
	XMLName xml.Name   `xml:"gpx"`
	Tracks  []gpxTrack `xml:"trk"`
	Routes  []gpxRoute `xml:"rte"`
}

type gpxTrack struct {
	Segments []gpxSegment `xml:"trkseg"`
}

type gpxSegment struct {
	Points []gpxPoint `xml:"trkpt"`
}

type gpxRoute struct {
	Points []gpxPoint `xml:"rtept"`
}

type gpxPoint struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Ele  float64 `xml:"ele"`
	Time string  `xml:"time"`
}

func parseGPX(r io.Reader) ([]Point, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read gpx: %w", err)
	}

	var gpx gpxFile
	if err := xml.Unmarshal(data, &gpx); err != nil {
		return nil, fmt.Errorf("parse gpx xml: %w", err)
	}

	var points []Point

	// Track points
	for _, trk := range gpx.Tracks {
		for _, seg := range trk.Segments {
			for _, pt := range seg.Points {
				points = append(points, gpxPointToPoint(pt))
			}
		}
	}

	// Route points (fallback if no tracks)
	for _, rte := range gpx.Routes {
		for _, pt := range rte.Points {
			points = append(points, gpxPointToPoint(pt))
		}
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("gpx file contains no track or route points")
	}

	return points, nil
}

func gpxPointToPoint(pt gpxPoint) Point {
	p := Point{
		Lat: pt.Lat,
		Lng: pt.Lon,
	}
	if pt.Ele != 0 {
		ele := pt.Ele
		p.Altitude = &ele
	}
	if pt.Time != "" {
		if t, err := time.Parse(time.RFC3339, pt.Time); err == nil {
			p.Time = &t
		} else if t, err := time.Parse("2006-01-02T15:04:05Z", pt.Time); err == nil {
			p.Time = &t
		} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", pt.Time); err == nil {
			p.Time = &t
		}
	}
	return p
}

// ---------------------------------------------------------------------------
// GeoJSON
// ---------------------------------------------------------------------------

type geoJSONFeature struct {
	Type     string          `json:"type"`
	Geometry geoJSONGeometry `json:"geometry"`
}

type geoJSONGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

func parseGeoJSON(r io.Reader) ([]Point, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read geojson: %w", err)
	}

	// Determine the root type
	var root struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse geojson: %w", err)
	}

	var geometries []geoJSONGeometry

	switch root.Type {
	case "FeatureCollection":
		var fc struct {
			Features []geoJSONFeature `json:"features"`
		}
		if err := json.Unmarshal(data, &fc); err != nil {
			return nil, fmt.Errorf("parse feature collection: %w", err)
		}
		for _, f := range fc.Features {
			geometries = append(geometries, f.Geometry)
		}

	case "Feature":
		var feat geoJSONFeature
		if err := json.Unmarshal(data, &feat); err != nil {
			return nil, fmt.Errorf("parse feature: %w", err)
		}
		geometries = append(geometries, feat.Geometry)

	default:
		// Bare geometry
		var geom geoJSONGeometry
		if err := json.Unmarshal(data, &geom); err != nil {
			return nil, fmt.Errorf("parse geometry: %w", err)
		}
		geometries = append(geometries, geom)
	}

	var points []Point
	for _, g := range geometries {
		pts, err := extractPoints(g)
		if err != nil {
			return nil, err
		}
		points = append(points, pts...)
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("geojson contains no extractable coordinates")
	}

	return points, nil
}

func extractPoints(g geoJSONGeometry) ([]Point, error) {
	switch g.Type {
	case "Point":
		var coord []float64
		if err := json.Unmarshal(g.Coordinates, &coord); err != nil {
			return nil, fmt.Errorf("parse Point coordinates: %w", err)
		}
		return []Point{coordToPoint(coord)}, nil

	case "MultiPoint", "LineString":
		var coords [][]float64
		if err := json.Unmarshal(g.Coordinates, &coords); err != nil {
			return nil, fmt.Errorf("parse %s coordinates: %w", g.Type, err)
		}
		pts := make([]Point, len(coords))
		for i, c := range coords {
			pts[i] = coordToPoint(c)
		}
		return pts, nil

	case "MultiLineString", "Polygon":
		var rings [][][]float64
		if err := json.Unmarshal(g.Coordinates, &rings); err != nil {
			return nil, fmt.Errorf("parse %s coordinates: %w", g.Type, err)
		}
		var pts []Point
		for _, ring := range rings {
			for _, c := range ring {
				pts = append(pts, coordToPoint(c))
			}
		}
		return pts, nil

	default:
		// Skip unsupported geometry types silently
		return nil, nil
	}
}

// coordToPoint converts a GeoJSON coordinate [lng, lat] or [lng, lat, alt] to a Point.
func coordToPoint(coord []float64) Point {
	p := Point{}
	if len(coord) >= 2 {
		p.Lng = coord[0]
		p.Lat = coord[1]
	}
	if len(coord) >= 3 {
		alt := coord[2]
		p.Altitude = &alt
	}
	return p
}
