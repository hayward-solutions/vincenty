// Package exif provides EXIF GPS extraction from image files.
package exif

import (
	"io"
	"time"

	exiflib "github.com/rwcarlsen/goexif/exif"
)

// Location holds GPS coordinates extracted from an image's EXIF data.
type Location struct {
	Lat      float64    `json:"lat"`
	Lng      float64    `json:"lng"`
	Altitude *float64   `json:"altitude,omitempty"`
	TakenAt  *time.Time `json:"taken_at,omitempty"`
}

// ExtractGPS attempts to read EXIF data from the given reader and extract
// GPS coordinates. Returns nil if no GPS data is found or if the file is
// not a valid EXIF image. The reader should be positioned at the start of
// the image data.
func ExtractGPS(r io.Reader) *Location {
	x, err := exiflib.Decode(r)
	if err != nil {
		return nil
	}

	lat, lng, err := x.LatLong()
	if err != nil {
		return nil
	}

	loc := &Location{
		Lat: lat,
		Lng: lng,
	}

	// Try to extract altitude
	if alt, err := x.Get(exiflib.GPSAltitude); err == nil {
		if val, err := alt.Rat(0); err == nil {
			f, _ := val.Float64()
			// Check altitude ref: 1 means below sea level
			if ref, err := x.Get(exiflib.GPSAltitudeRef); err == nil {
				if v, err := ref.Int(0); err == nil && v == 1 {
					f = -f
				}
			}
			loc.Altitude = &f
		}
	}

	// Try to extract original date/time
	if dt, err := x.DateTime(); err == nil {
		loc.TakenAt = &dt
	}

	return loc
}
