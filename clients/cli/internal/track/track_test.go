package track

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// GPX parsing
// ---------------------------------------------------------------------------

const testGPX = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <trk>
    <trkseg>
      <trkpt lat="-33.8688" lon="151.2093">
        <ele>10.5</ele>
        <time>2024-01-01T00:00:00Z</time>
      </trkpt>
      <trkpt lat="-33.8700" lon="151.2100">
        <ele>12.0</ele>
        <time>2024-01-01T00:00:10Z</time>
      </trkpt>
      <trkpt lat="-33.8712" lon="151.2110">
        <time>2024-01-01T00:00:20Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseGPX_TrackPoints(t *testing.T) {
	path := writeTemp(t, "test.gpx", testGPX)
	points, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(points) != 3 {
		t.Fatalf("len = %d, want 3", len(points))
	}

	// First point
	if points[0].Lat != -33.8688 {
		t.Errorf("points[0].Lat = %f", points[0].Lat)
	}
	if points[0].Lng != 151.2093 {
		t.Errorf("points[0].Lng = %f", points[0].Lng)
	}
	if points[0].Altitude == nil || *points[0].Altitude != 10.5 {
		t.Errorf("points[0].Altitude = %v", points[0].Altitude)
	}
	if points[0].Time == nil {
		t.Fatal("points[0].Time should not be nil")
	}
	expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !points[0].Time.Equal(expected) {
		t.Errorf("points[0].Time = %v, want %v", points[0].Time, expected)
	}

	// Third point has no elevation
	if points[2].Altitude != nil {
		t.Errorf("points[2].Altitude = %v, want nil", points[2].Altitude)
	}
}

const testGPXRoute = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <rte>
    <rtept lat="40.7128" lon="-74.0060"/>
    <rtept lat="40.7580" lon="-73.9855"/>
  </rte>
</gpx>`

func TestParseGPX_RoutePoints(t *testing.T) {
	path := writeTemp(t, "route.gpx", testGPXRoute)
	points, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("len = %d, want 2", len(points))
	}
	if points[0].Lat != 40.7128 {
		t.Errorf("points[0].Lat = %f", points[0].Lat)
	}
}

const testGPXEmpty = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test"></gpx>`

func TestParseGPX_Empty(t *testing.T) {
	path := writeTemp(t, "empty.gpx", testGPXEmpty)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for empty GPX")
	}
}

// ---------------------------------------------------------------------------
// GeoJSON parsing
// ---------------------------------------------------------------------------

const testGeoJSONLineString = `{
  "type": "FeatureCollection",
  "features": [{
    "type": "Feature",
    "geometry": {
      "type": "LineString",
      "coordinates": [
        [151.2093, -33.8688, 10.5],
        [151.2100, -33.8700],
        [151.2110, -33.8712, 15.0]
      ]
    }
  }]
}`

func TestParseGeoJSON_FeatureCollectionLineString(t *testing.T) {
	path := writeTemp(t, "test.geojson", testGeoJSONLineString)
	points, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(points) != 3 {
		t.Fatalf("len = %d, want 3", len(points))
	}

	// GeoJSON is [lng, lat] order
	if points[0].Lat != -33.8688 {
		t.Errorf("points[0].Lat = %f", points[0].Lat)
	}
	if points[0].Lng != 151.2093 {
		t.Errorf("points[0].Lng = %f", points[0].Lng)
	}
	if points[0].Altitude == nil || *points[0].Altitude != 10.5 {
		t.Errorf("points[0].Altitude = %v", points[0].Altitude)
	}
	// Second point has no altitude
	if points[1].Altitude != nil {
		t.Errorf("points[1].Altitude should be nil")
	}
	// GeoJSON has no timestamps
	if points[0].Time != nil {
		t.Error("GeoJSON points should have nil Time")
	}
}

const testGeoJSONSingleFeature = `{
  "type": "Feature",
  "geometry": {
    "type": "LineString",
    "coordinates": [[0.0, 1.0], [2.0, 3.0]]
  }
}`

func TestParseGeoJSON_SingleFeature(t *testing.T) {
	path := writeTemp(t, "single.geojson", testGeoJSONSingleFeature)
	points, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("len = %d, want 2", len(points))
	}
}

const testGeoJSONBareGeometry = `{
  "type": "LineString",
  "coordinates": [[10.0, 20.0], [30.0, 40.0]]
}`

func TestParseGeoJSON_BareGeometry(t *testing.T) {
	path := writeTemp(t, "bare.json", testGeoJSONBareGeometry)
	points, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("len = %d, want 2", len(points))
	}
	if points[0].Lng != 10.0 || points[0].Lat != 20.0 {
		t.Errorf("points[0] = {%f, %f}", points[0].Lng, points[0].Lat)
	}
}

const testGeoJSONPoint = `{
  "type": "Feature",
  "geometry": {
    "type": "Point",
    "coordinates": [151.2093, -33.8688]
  }
}`

func TestParseGeoJSON_Point(t *testing.T) {
	path := writeTemp(t, "point.geojson", testGeoJSONPoint)
	points, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("len = %d, want 1", len(points))
	}
}

const testGeoJSONMultiLineString = `{
  "type": "Feature",
  "geometry": {
    "type": "MultiLineString",
    "coordinates": [
      [[0, 1], [2, 3]],
      [[4, 5], [6, 7]]
    ]
  }
}`

func TestParseGeoJSON_MultiLineString(t *testing.T) {
	path := writeTemp(t, "multi.geojson", testGeoJSONMultiLineString)
	points, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 4 {
		t.Fatalf("len = %d, want 4", len(points))
	}
}

func TestParseGeoJSON_Empty(t *testing.T) {
	path := writeTemp(t, "empty.geojson", `{"type":"FeatureCollection","features":[]}`)
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for empty GeoJSON")
	}
}

// ---------------------------------------------------------------------------
// Unsupported extension
// ---------------------------------------------------------------------------

func TestLoad_UnsupportedExtension(t *testing.T) {
	path := writeTemp(t, "test.kml", "<kml/>")
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for unsupported extension")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/file.gpx")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// ---------------------------------------------------------------------------
// Heading calculation
// ---------------------------------------------------------------------------

func TestHeading_NorthEast(t *testing.T) {
	// Sydney to a point due north-east
	p1 := Point{Lat: -33.8688, Lng: 151.2093}
	p2 := Point{Lat: -33.8600, Lng: 151.2200}

	h := Heading(p1, p2)
	if h == nil {
		t.Fatal("heading should not be nil")
	}
	// Should be roughly NE (30-60 degrees)
	if *h < 20 || *h > 70 {
		t.Errorf("heading = %f, expected roughly NE", *h)
	}
}

func TestHeading_DueEast(t *testing.T) {
	p1 := Point{Lat: 0, Lng: 0}
	p2 := Point{Lat: 0, Lng: 1}

	h := Heading(p1, p2)
	if h == nil {
		t.Fatal("heading should not be nil")
	}
	if math.Abs(*h-90) > 0.1 {
		t.Errorf("heading = %f, want ~90", *h)
	}
}

// ---------------------------------------------------------------------------
// Speed calculation
// ---------------------------------------------------------------------------

func TestSpeed_WithTimestamps(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 0, 0, 10, 0, time.UTC)

	// Two points ~157m apart at the equator (0.001 degree longitude)
	p1 := Point{Lat: 0, Lng: 0, Time: &t1}
	p2 := Point{Lat: 0, Lng: 0.001, Time: &t2}

	s := Speed(p1, p2)
	if s == nil {
		t.Fatal("speed should not be nil")
	}
	// ~111m / 10s ≈ 11.1 m/s
	if *s < 5 || *s > 20 {
		t.Errorf("speed = %f m/s, expected ~11.1", *s)
	}
}

func TestSpeed_NoTimestamps(t *testing.T) {
	p1 := Point{Lat: 0, Lng: 0}
	p2 := Point{Lat: 1, Lng: 1}

	s := Speed(p1, p2)
	if s != nil {
		t.Error("speed should be nil without timestamps")
	}
}

func TestSpeed_SameTime(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	p1 := Point{Lat: 0, Lng: 0, Time: &t1}
	p2 := Point{Lat: 1, Lng: 1, Time: &t1}

	s := Speed(p1, p2)
	if s != nil {
		t.Error("speed should be nil when timestamps are equal")
	}
}
