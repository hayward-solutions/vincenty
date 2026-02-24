package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/pubsub"
	"github.com/sitaware/api/internal/repository"
	mockrepo "github.com/sitaware/api/internal/repository/mock"
)

func TestLocationService_Update_Accepted(t *testing.T) {
	persisted := false
	deviceUpdated := false

	locRepo := &mockrepo.LocationRepo{
		CreateFn: func(ctx context.Context, userID, deviceID uuid.UUID, lat, lng float64, altitude, heading, speed, accuracy *float64) error {
			persisted = true
			return nil
		},
		UpdateDeviceLocationFn: func(ctx context.Context, deviceID uuid.UUID, lat, lng float64) error {
			deviceUpdated = true
			return nil
		},
	}
	ps := pubsub.NewMockPubSub()
	svc := NewLocationService(locRepo, nil, ps, 0) // no throttle

	groups := []uuid.UUID{uuid.New(), uuid.New()}
	accepted, err := svc.Update(context.Background(),
		uuid.New(), uuid.New(), "bob", "Bob", "iPhone", true,
		-33.86, 151.20, nil, nil, nil, nil,
		groups,
	)

	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !accepted {
		t.Error("expected accepted=true")
	}
	if !persisted {
		t.Error("expected location to be persisted")
	}
	if !deviceUpdated {
		t.Error("expected device location to be updated")
	}
	published := len(ps.Published())
	if published != 2 {
		t.Errorf("published = %d, want 2 (one per group)", published)
	}
}

func TestLocationService_Update_Throttled(t *testing.T) {
	locRepo := &mockrepo.LocationRepo{
		CreateFn: func(ctx context.Context, userID, deviceID uuid.UUID, lat, lng float64, altitude, heading, speed, accuracy *float64) error {
			return nil
		},
		UpdateDeviceLocationFn: func(ctx context.Context, deviceID uuid.UUID, lat, lng float64) error {
			return nil
		},
	}
	ps := pubsub.NewMockPubSub()
	svc := NewLocationService(locRepo, nil, ps, 10*time.Second) // 10s throttle
	deviceID := uuid.New()
	groups := []uuid.UUID{uuid.New()}

	// First call should be accepted
	accepted, _ := svc.Update(context.Background(),
		uuid.New(), deviceID, "bob", "Bob", "iPhone", true,
		-33.86, 151.20, nil, nil, nil, nil, groups)
	if !accepted {
		t.Error("first call should be accepted")
	}

	// Immediate second call should be throttled
	accepted, err := svc.Update(context.Background(),
		uuid.New(), deviceID, "bob", "Bob", "iPhone", true,
		-33.87, 151.21, nil, nil, nil, nil, groups)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if accepted {
		t.Error("second call should be throttled")
	}
}

func TestLocationService_Update_DifferentDevicesNotThrottled(t *testing.T) {
	callCount := 0
	locRepo := &mockrepo.LocationRepo{
		CreateFn: func(ctx context.Context, userID, deviceID uuid.UUID, lat, lng float64, altitude, heading, speed, accuracy *float64) error {
			callCount++
			return nil
		},
		UpdateDeviceLocationFn: func(ctx context.Context, deviceID uuid.UUID, lat, lng float64) error {
			return nil
		},
	}
	ps := pubsub.NewMockPubSub()
	svc := NewLocationService(locRepo, nil, ps, 10*time.Second)

	svc.Update(context.Background(),
		uuid.New(), uuid.New(), "bob", "Bob", "iPhone", true,
		0, 0, nil, nil, nil, nil, nil)
	svc.Update(context.Background(),
		uuid.New(), uuid.New(), "alice", "Alice", "Pixel", true,
		0, 0, nil, nil, nil, nil, nil)

	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (different devices)", callCount)
	}
}

func TestLocationService_GetGroupSnapshot(t *testing.T) {
	groupID := uuid.New()
	expected := []repository.LocationRecord{{Lat: 1, Lng: 2}}

	locRepo := &mockrepo.LocationRepo{
		GetLatestByGroupFn: func(ctx context.Context, gid uuid.UUID) ([]repository.LocationRecord, error) {
			if gid != groupID {
				t.Errorf("GetLatestByGroup called with %v, want %v", gid, groupID)
			}
			return expected, nil
		},
	}
	svc := NewLocationService(locRepo, nil, nil, 0)

	records, err := svc.GetGroupSnapshot(context.Background(), groupID)
	if err != nil {
		t.Fatalf("GetGroupSnapshot() error = %v", err)
	}
	if len(records) != 1 || records[0].Lat != 1 {
		t.Errorf("unexpected records: %v", records)
	}
}

func TestLocationService_GetGroupHistory_Admin(t *testing.T) {
	groupID := uuid.New()
	locRepo := &mockrepo.LocationRepo{
		GetGroupHistoryFn: func(ctx context.Context, gid uuid.UUID, from, to time.Time) ([]repository.LocationRecord, error) {
			return []repository.LocationRecord{{Lat: 10, Username: "bob"}}, nil
		},
	}
	svc := NewLocationService(locRepo, nil, nil, 0)

	entries, err := svc.GetGroupHistory(context.Background(), groupID, uuid.New(), true, time.Now().Add(-1*time.Hour), time.Now())
	if err != nil {
		t.Fatalf("GetGroupHistory() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Username != "bob" {
		t.Errorf("unexpected entries: %v", entries)
	}
}

func TestLocationService_GetGroupHistory_NonMember_Forbidden(t *testing.T) {
	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return nil, model.ErrNotFound("group member")
		},
	}
	svc := NewLocationService(nil, groupRepo, nil, 0)

	_, err := svc.GetGroupHistory(context.Background(), uuid.New(), uuid.New(), false, time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected error for non-member")
	}
}

func TestLocationService_GetVisibleHistory_Admin(t *testing.T) {
	locRepo := &mockrepo.LocationRepo{
		GetAllHistoryFn: func(ctx context.Context, from, to time.Time) ([]repository.LocationRecord, error) {
			return []repository.LocationRecord{{Lat: 1}}, nil
		},
	}
	svc := NewLocationService(locRepo, nil, nil, 0)

	entries, err := svc.GetVisibleHistory(context.Background(), uuid.New(), true, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("GetVisibleHistory() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
}

func TestLocationService_GetVisibleHistory_NonAdmin(t *testing.T) {
	callerID := uuid.New()
	locRepo := &mockrepo.LocationRepo{
		GetVisibleHistoryFn: func(ctx context.Context, cid uuid.UUID, from, to time.Time) ([]repository.LocationRecord, error) {
			if cid != callerID {
				t.Errorf("expected callerID %v, got %v", callerID, cid)
			}
			return nil, nil
		},
	}
	svc := NewLocationService(locRepo, nil, nil, 0)

	_, err := svc.GetVisibleHistory(context.Background(), callerID, false, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("GetVisibleHistory() error = %v", err)
	}
}

func TestLocationService_GetUserHistory_Self(t *testing.T) {
	callerID := uuid.New()
	locRepo := &mockrepo.LocationRepo{
		GetUserHistoryFn: func(ctx context.Context, userID uuid.UUID, from, to time.Time, deviceID *uuid.UUID) ([]repository.LocationRecord, error) {
			return []repository.LocationRecord{{Lat: 5}}, nil
		},
	}
	svc := NewLocationService(locRepo, nil, nil, 0)

	entries, err := svc.GetUserHistory(context.Background(), callerID, callerID, false, time.Now(), time.Now(), nil)
	if err != nil {
		t.Fatalf("GetUserHistory() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
}

func TestLocationService_GetUserHistory_NoSharedGroup_Forbidden(t *testing.T) {
	locRepo := &mockrepo.LocationRepo{
		UsersShareGroupFn: func(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
			return false, nil
		},
	}
	svc := NewLocationService(locRepo, nil, nil, 0)

	_, err := svc.GetUserHistory(context.Background(), uuid.New(), uuid.New(), false, time.Now(), time.Now(), nil)
	if err == nil {
		t.Fatal("expected error when users don't share a group")
	}
}

func TestLocationService_GetAllLatest(t *testing.T) {
	dn := "Bob Smith"
	devName := "iPhone"
	locRepo := &mockrepo.LocationRepo{
		GetAllLatestFn: func(ctx context.Context) ([]repository.LocationRecord, error) {
			return []repository.LocationRecord{
				{Lat: 1, Lng: 2, Username: "bob", DisplayName: &dn, DeviceName: &devName, IsPrimary: true},
			}, nil
		},
	}
	svc := NewLocationService(locRepo, nil, nil, 0)

	entries, err := svc.GetAllLatest(context.Background())
	if err != nil {
		t.Fatalf("GetAllLatest() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].DisplayName != "Bob Smith" {
		t.Errorf("DisplayName = %q, want %q", entries[0].DisplayName, "Bob Smith")
	}
	if entries[0].DeviceName != "iPhone" {
		t.Errorf("DeviceName = %q, want %q", entries[0].DeviceName, "iPhone")
	}
	if !entries[0].IsPrimary {
		t.Error("expected IsPrimary=true")
	}
}

func TestToHistoryEntry(t *testing.T) {
	dn := "Alice"
	devName := "Pixel"
	alt := 100.0
	rec := repository.LocationRecord{
		UserID:      uuid.New(),
		DeviceID:    uuid.New(),
		Lat:         -33.86,
		Lng:         151.20,
		Altitude:    &alt,
		Username:    "alice",
		DisplayName: &dn,
		DeviceName:  &devName,
		RecordedAt:  time.Now(),
	}

	entry := toHistoryEntry(rec)

	if entry.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want %q", entry.DisplayName, "Alice")
	}
	if entry.DeviceName != "Pixel" {
		t.Errorf("DeviceName = %q, want %q", entry.DeviceName, "Pixel")
	}
	if entry.Altitude == nil || *entry.Altitude != 100.0 {
		t.Errorf("Altitude = %v, want 100.0", entry.Altitude)
	}
}

func TestToHistoryEntry_NilOptionals(t *testing.T) {
	rec := repository.LocationRecord{
		Lat:      0,
		Lng:      0,
		Username: "test",
	}

	entry := toHistoryEntry(rec)

	if entry.DisplayName != "" {
		t.Errorf("DisplayName = %q, want empty", entry.DisplayName)
	}
	if entry.DeviceName != "" {
		t.Errorf("DeviceName = %q, want empty", entry.DeviceName)
	}
}
