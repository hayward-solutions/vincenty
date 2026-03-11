package ws

import (
	"encoding/json"
	"testing"
)

func TestNewEnvelope(t *testing.T) {
	payload := struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}{Lat: -33.86, Lng: 151.20}

	data, err := NewEnvelope(TypeLocationBroadcast, payload)
	if err != nil {
		t.Fatalf("NewEnvelope() error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("Unmarshal envelope error: %v", err)
	}
	if env.Type != TypeLocationBroadcast {
		t.Errorf("Type = %q, want %q", env.Type, TypeLocationBroadcast)
	}

	var inner struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	if err := json.Unmarshal(env.Payload, &inner); err != nil {
		t.Fatalf("Unmarshal payload error: %v", err)
	}
	if inner.Lat != -33.86 || inner.Lng != 151.20 {
		t.Errorf("Payload = {%v, %v}, want {-33.86, 151.20}", inner.Lat, inner.Lng)
	}
}

func TestNewEnvelope_AllTypes(t *testing.T) {
	types := []string{
		TypeLocationUpdate,
		TypeLocationBroadcast,
		TypeLocationSnapshot,
		TypeMessageNew,
		TypeDrawingUpdated,
		TypeConnected,
		TypeError,
	}

	for _, msgType := range types {
		t.Run(msgType, func(t *testing.T) {
			data, err := NewEnvelope(msgType, map[string]string{"key": "value"})
			if err != nil {
				t.Fatalf("NewEnvelope(%q) error: %v", msgType, err)
			}

			var env Envelope
			if err := json.Unmarshal(data, &env); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if env.Type != msgType {
				t.Errorf("Type = %q, want %q", env.Type, msgType)
			}
		})
	}
}

func TestNewEnvelope_UnmarshalablePayload(t *testing.T) {
	// Channels cannot be marshalled to JSON
	_, err := NewEnvelope("test", make(chan int))
	if err == nil {
		t.Error("expected error for unmarshalable payload, got nil")
	}
}

func TestNewEnvelope_NilPayload(t *testing.T) {
	data, err := NewEnvelope("test", nil)
	if err != nil {
		t.Fatalf("NewEnvelope with nil payload error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if string(env.Payload) != "null" {
		t.Errorf("Payload = %s, want null", string(env.Payload))
	}
}

func TestEnvelope_RoundTrip(t *testing.T) {
	orig := ErrorPayload{Message: "something went wrong"}
	data, err := NewEnvelope(TypeError, orig)
	if err != nil {
		t.Fatalf("NewEnvelope error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("Unmarshal envelope: %v", err)
	}

	var payload ErrorPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal payload: %v", err)
	}
	if payload.Message != "something went wrong" {
		t.Errorf("Message = %q, want %q", payload.Message, "something went wrong")
	}
}

func TestLocationUpdatePayload_JSON(t *testing.T) {
	alt := 50.0
	speed := 3.0
	payload := LocationUpdatePayload{
		DeviceID: "device-123",
		Lat:      -33.86,
		Lng:      151.20,
		Altitude: &alt,
		Speed:    &speed,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded LocationUpdatePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.DeviceID != "device-123" {
		t.Errorf("DeviceID = %q, want %q", decoded.DeviceID, "device-123")
	}
	if decoded.Altitude == nil || *decoded.Altitude != 50.0 {
		t.Errorf("Altitude = %v, want 50.0", decoded.Altitude)
	}
	if decoded.Heading != nil {
		t.Errorf("Heading should be nil (omitempty), got %v", *decoded.Heading)
	}
}

func TestConnectedPayload_JSON(t *testing.T) {
	payload := ConnectedPayload{
		Groups: []ConnectedGroup{
			{Name: "Team Alpha"},
			{Name: "Team Bravo"},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ConnectedPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(decoded.Groups) != 2 {
		t.Fatalf("Groups count = %d, want 2", len(decoded.Groups))
	}
	if decoded.Groups[0].Name != "Team Alpha" {
		t.Errorf("Groups[0].Name = %q, want %q", decoded.Groups[0].Name, "Team Alpha")
	}
}

func TestTypeConstants(t *testing.T) {
	// Ensure type constants are non-empty and unique
	types := map[string]bool{}
	all := []string{
		TypeLocationUpdate,
		TypeLocationBroadcast,
		TypeLocationSnapshot,
		TypeMessageNew,
		TypeDrawingUpdated,
		TypeConnected,
		TypeError,
	}
	for _, typ := range all {
		if typ == "" {
			t.Error("found empty type constant")
		}
		if types[typ] {
			t.Errorf("duplicate type constant: %q", typ)
		}
		types[typ] = true
	}
}
