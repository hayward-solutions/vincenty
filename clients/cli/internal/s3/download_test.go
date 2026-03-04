package s3

import "testing"

func TestIsS3URI(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"s3://bucket/key.gpx", true},
		{"s3://my-bucket/path/to/file.gpx", true},
		{"s3://b/k", true},
		{"/local/path.gpx", false},
		{"track.gpx", false},
		{"", false},
		{"S3://bucket/key", false}, // case-sensitive
		{"s3:", false},
		{"s3:/", false},
	}
	for _, tt := range tests {
		if got := IsS3URI(tt.path); got != tt.want {
			t.Errorf("IsS3URI(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestParseURI(t *testing.T) {
	tests := []struct {
		uri        string
		wantBucket string
		wantKey    string
		wantErr    bool
	}{
		{"s3://my-bucket/path/to/file.gpx", "my-bucket", "path/to/file.gpx", false},
		{"s3://bucket/key.gpx", "bucket", "key.gpx", false},
		{"s3://bucket/a/b/c/d.geojson", "bucket", "a/b/c/d.geojson", false},
		{"s3://bucket/user-123/activity-456.gpx", "bucket", "user-123/activity-456.gpx", false},

		// Invalid URIs
		{"s3://bucket/", "", "", true},  // trailing slash, empty key
		{"s3://bucket", "", "", true},   // no key at all
		{"s3:///key", "", "", true},     // empty bucket
		{"s3://", "", "", true},         // empty bucket and key
		{"/local/path", "", "", true},   // not an S3 URI
		{"", "", "", true},              // empty string
	}
	for _, tt := range tests {
		bucket, key, err := ParseURI(tt.uri)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseURI(%q) error = %v, wantErr %v", tt.uri, err, tt.wantErr)
			continue
		}
		if !tt.wantErr {
			if bucket != tt.wantBucket {
				t.Errorf("ParseURI(%q) bucket = %q, want %q", tt.uri, bucket, tt.wantBucket)
			}
			if key != tt.wantKey {
				t.Errorf("ParseURI(%q) key = %q, want %q", tt.uri, key, tt.wantKey)
			}
		}
	}
}
