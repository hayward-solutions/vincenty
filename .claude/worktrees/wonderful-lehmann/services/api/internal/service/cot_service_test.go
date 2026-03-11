package service

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/pubsub"
	"github.com/vincenty/api/internal/repository"
	mockrepo "github.com/vincenty/api/internal/repository/mock"
)

const validCotXML = `<?xml version="1.0" encoding="UTF-8"?>
<event version="2.0" uid="ANDROID-abc123" type="a-f-G-U-C" time="2024-01-15T10:30:00Z" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="m-g">
  <point lat="-33.8688" lon="151.2093" hae="50.5" ce="10.0" le="5.0"/>
  <detail>
    <contact callsign="Alpha1"/>
    <track speed="1.5" course="270.0"/>
  </detail>
</event>`

func TestCotService_Ingest_StoresEvent(t *testing.T) {
	cotStored := false
	cotRepo := &mockrepo.CotRepo{
		CreateFn: func(ctx context.Context, evt *model.CotEvent) error {
			cotStored = true
			if evt.EventUID != "ANDROID-abc123" {
				t.Errorf("EventUID = %q, want %q", evt.EventUID, "ANDROID-abc123")
			}
			return nil
		},
	}
	deviceRepo := &mockrepo.DeviceRepo{
		GetByDeviceUIDFn: func(ctx context.Context, deviceUID string) (*model.Device, error) {
			return nil, model.ErrNotFound("device")
		},
	}
	svc := NewCotService(cotRepo, deviceRepo, nil, nil, nil)

	result, err := svc.Ingest(context.Background(), strings.NewReader(validCotXML))
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if !cotStored {
		t.Error("expected event to be stored")
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if result.Stored != 1 {
		t.Errorf("Stored = %d, want 1", result.Stored)
	}
}

func TestCotService_Ingest_WithDeviceMapping(t *testing.T) {
	userID := uuid.New()
	deviceID := uuid.New()

	cotRepo := &mockrepo.CotRepo{
		CreateFn: func(ctx context.Context, evt *model.CotEvent) error {
			if evt.UserID == nil || *evt.UserID != userID {
				t.Errorf("expected UserID %v, got %v", userID, evt.UserID)
			}
			return nil
		},
	}
	deviceRepo := &mockrepo.DeviceRepo{
		GetByDeviceUIDFn: func(ctx context.Context, deviceUID string) (*model.Device, error) {
			return &model.Device{ID: deviceID, UserID: userID, Name: "ATAK"}, nil
		},
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
			return &model.Device{ID: deviceID, UserID: userID, Name: "ATAK", IsPrimary: true}, nil
		},
	}
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, Username: "alpha1"}, nil
		},
	}
	groupRepo := &mockrepo.GroupRepo{
		ListByUserIDFn: func(ctx context.Context, uid uuid.UUID) ([]model.Group, []int, error) {
			return []model.Group{{ID: uuid.New(), Name: "Team"}}, []int{5}, nil
		},
	}

	locRepo := &mockrepo.LocationRepo{
		CreateFn: func(ctx context.Context, uid, did uuid.UUID, lat, lng float64, altitude, heading, speed, accuracy *float64) error {
			return nil
		},
		UpdateDeviceLocationFn: func(ctx context.Context, did uuid.UUID, lat, lng float64) error {
			return nil
		},
	}
	ps := pubsub.NewMockPubSub()
	locationSvc := NewLocationService(locRepo, nil, ps, 0)

	svc := NewCotService(cotRepo, deviceRepo, userRepo, groupRepo, locationSvc)

	result, err := svc.Ingest(context.Background(), strings.NewReader(validCotXML))
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if result.Bridged != 1 {
		t.Errorf("Bridged = %d, want 1", result.Bridged)
	}
}

func TestCotService_Ingest_InvalidXML(t *testing.T) {
	svc := NewCotService(nil, nil, nil, nil, nil)
	_, err := svc.Ingest(context.Background(), strings.NewReader("not xml"))
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}

func TestCotService_List(t *testing.T) {
	expected := []model.CotEvent{{EventUID: "test"}}
	cotRepo := &mockrepo.CotRepo{
		ListFn: func(ctx context.Context, f model.CotEventFilters) ([]model.CotEvent, int, error) {
			return expected, 1, nil
		},
	}
	svc := NewCotService(cotRepo, nil, nil, nil, nil)

	events, total, err := svc.List(context.Background(), model.CotEventFilters{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(events) != 1 {
		t.Errorf("unexpected: events=%d, total=%d", len(events), total)
	}
}

func TestCotService_GetLatestByUID(t *testing.T) {
	expected := &model.CotEvent{EventUID: "TEST-123"}
	cotRepo := &mockrepo.CotRepo{
		GetLatestByUIDFn: func(ctx context.Context, eventUID string) (*model.CotEvent, error) {
			return expected, nil
		},
	}
	svc := NewCotService(cotRepo, nil, nil, nil, nil)

	evt, err := svc.GetLatestByUID(context.Background(), "TEST-123")
	if err != nil {
		t.Fatalf("GetLatestByUID() error = %v", err)
	}
	if evt.EventUID != "TEST-123" {
		t.Errorf("EventUID = %q, want %q", evt.EventUID, "TEST-123")
	}
}

func TestCotService_ResolveUserInfo(t *testing.T) {
	userID := uuid.New()
	dn := "Bob Smith"
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, Username: "bob", DisplayName: &dn}, nil
		},
	}
	svc := NewCotService(nil, nil, userRepo, nil, nil)

	username, displayName := svc.resolveUserInfo(context.Background(), userID)
	if username != "bob" {
		t.Errorf("username = %q, want %q", username, "bob")
	}
	if displayName != "Bob Smith" {
		t.Errorf("displayName = %q, want %q", displayName, "Bob Smith")
	}
}

func TestCotService_ResolveUserInfo_NotFound(t *testing.T) {
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return nil, model.ErrNotFound("user")
		},
	}
	svc := NewCotService(nil, nil, userRepo, nil, nil)

	username, displayName := svc.resolveUserInfo(context.Background(), uuid.New())
	if username != "" || displayName != "" {
		t.Errorf("expected empty strings, got %q, %q", username, displayName)
	}
}

func TestCotService_Ingest_BatchEvents(t *testing.T) {
	batchXML := `<events>
<event version="2.0" uid="UID1" type="a-f-G" time="2024-01-15T10:30:00Z" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="m-g">
  <point lat="10" lon="20" hae="0" ce="0" le="0"/>
  <detail/>
</event>
<event version="2.0" uid="UID2" type="b-m-p-w" time="2024-01-15T10:30:00Z" start="2024-01-15T10:30:00Z" stale="2024-01-15T10:35:00Z" how="h-e">
  <point lat="30" lon="40" hae="0" ce="0" le="0"/>
  <detail/>
</event>
</events>`

	stored := 0
	cotRepo := &mockrepo.CotRepo{
		CreateFn: func(ctx context.Context, evt *model.CotEvent) error {
			stored++
			return nil
		},
	}
	deviceRepo := &mockrepo.DeviceRepo{
		GetByDeviceUIDFn: func(ctx context.Context, deviceUID string) (*model.Device, error) {
			return nil, model.ErrNotFound("device")
		},
	}
	locRepo := &mockrepo.LocationRepo{}
	ps := pubsub.NewMockPubSub()
	locationSvc := NewLocationService(locRepo, nil, ps, 0)

	svc := NewCotService(cotRepo, deviceRepo, nil, nil, locationSvc)
	_ = repository.LocationRecord{} // ensure import used

	result, err := svc.Ingest(context.Background(), strings.NewReader(batchXML))
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if stored != 2 {
		t.Errorf("stored = %d, want 2", stored)
	}
}
