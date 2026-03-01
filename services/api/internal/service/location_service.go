package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/pubsub"
	"github.com/sitaware/api/internal/repository"
)

// LocationBroadcast is the JSON payload published to Redis for location updates.
type LocationBroadcast struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	DeviceID    uuid.UUID `json:"device_id"`
	DeviceName  string    `json:"device_name"`
	IsPrimary   bool      `json:"is_primary"`
	GroupID     uuid.UUID `json:"group_id"`
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	Altitude    *float64  `json:"altitude,omitempty"`
	Heading     *float64  `json:"heading,omitempty"`
	Speed       *float64  `json:"speed,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// LocationService handles location update logic including throttling,
// persistence, and pub/sub broadcasting.
type LocationService struct {
	locationRepo repository.LocationRepo
	groupRepo    repository.GroupRepo
	ps           pubsub.PubSub
	throttle     time.Duration

	// Per-device throttle tracking
	lastUpdate sync.Map // map[uuid.UUID]time.Time
}

// NewLocationService creates a new LocationService.
func NewLocationService(
	locationRepo repository.LocationRepo,
	groupRepo repository.GroupRepo,
	ps pubsub.PubSub,
	throttle time.Duration,
) *LocationService {
	return &LocationService{
		locationRepo: locationRepo,
		groupRepo:    groupRepo,
		ps:           ps,
		throttle:     throttle,
	}
}

// Update processes a location update: throttle → persist → broadcast.
// Returns true if the update was accepted, false if throttled.
func (s *LocationService) Update(
	ctx context.Context,
	userID, deviceID uuid.UUID,
	username, displayName, deviceName string,
	isPrimary bool,
	lat, lng float64,
	altitude, heading, speed, accuracy *float64,
	groups []uuid.UUID,
) (bool, error) {
	slog.Debug("location service: Update called",
		"user_id", userID,
		"device_id", deviceID,
		"lat", lat,
		"lng", lng,
		"groups", len(groups),
	)

	// --- Throttle check ---
	now := time.Now()
	if last, ok := s.lastUpdate.Load(deviceID); ok {
		if elapsed := now.Sub(last.(time.Time)); elapsed < s.throttle {
			slog.Debug("location service: throttled",
				"device_id", deviceID,
				"elapsed", elapsed,
				"throttle", s.throttle,
			)
			return false, nil // throttled
		}
	}
	s.lastUpdate.Store(deviceID, now)

	// --- Persist to location_history ---
	if err := s.locationRepo.Create(ctx, userID, deviceID, lat, lng, altitude, heading, speed, accuracy); err != nil {
		slog.Error("failed to insert location history", "error", err, "user_id", userID)
		return false, err
	}
	slog.Debug("location service: persisted to location_history",
		"user_id", userID,
		"device_id", deviceID,
	)

	// --- Update device's last known location ---
	if err := s.locationRepo.UpdateDeviceLocation(ctx, deviceID, lat, lng); err != nil {
		slog.Error("failed to update device location", "error", err, "device_id", deviceID)
		// Non-fatal: continue with broadcast
	} else {
		slog.Debug("location service: updated device last_location",
			"device_id", deviceID,
		)
	}

	// --- Broadcast to all user's groups ---
	if len(groups) == 0 {
		slog.Warn("location service: no groups to broadcast to — update persisted but no real-time broadcast will occur",
			"user_id", userID,
			"device_id", deviceID,
		)
	}

	broadcast := LocationBroadcast{
		UserID:      userID,
		Username:    username,
		DisplayName: displayName,
		DeviceID:    deviceID,
		DeviceName:  deviceName,
		IsPrimary:   isPrimary,
		Lat:         lat,
		Lng:         lng,
		Altitude:    altitude,
		Heading:     heading,
		Speed:       speed,
		Timestamp:   now,
	}

	for _, gid := range groups {
		broadcast.GroupID = gid
		data, err := json.Marshal(broadcast)
		if err != nil {
			slog.Error("failed to marshal location broadcast", "error", err)
			continue
		}
		channel := fmt.Sprintf("group:%s:location", gid)
		if err := s.ps.Publish(ctx, channel, data); err != nil {
			slog.Error("failed to publish location", "error", err, "channel", channel)
		} else {
			slog.Debug("location service: published to Redis",
				"channel", channel,
				"user_id", userID,
			)
		}
	}

	// Also publish to the user's personal channel so their other connected
	// devices receive the update even when they share no common group.
	broadcast.GroupID = uuid.Nil
	if selfData, err := json.Marshal(broadcast); err != nil {
		slog.Error("failed to marshal self location broadcast", "error", err)
	} else {
		selfChannel := fmt.Sprintf("user:%s:location", userID)
		if err := s.ps.Publish(ctx, selfChannel, selfData); err != nil {
			slog.Error("failed to publish self location", "error", err, "channel", selfChannel)
		} else {
			slog.Debug("location service: published to user self-channel",
				"channel", selfChannel,
				"user_id", userID,
			)
		}
	}

	return true, nil
}

// GetUserSnapshot returns the latest position per device for a specific user.
// Used to populate a self-snapshot on WebSocket connect.
func (s *LocationService) GetUserSnapshot(ctx context.Context, userID uuid.UUID) ([]repository.LocationRecord, error) {
	return s.locationRepo.GetLatestByUser(ctx, userID)
}

// GetGroupSnapshot returns the latest position per user in a group.
func (s *LocationService) GetGroupSnapshot(ctx context.Context, groupID uuid.UUID) ([]repository.LocationRecord, error) {
	return s.locationRepo.GetLatestByGroup(ctx, groupID)
}

// LocationHistoryEntry is a single point in a location track.
type LocationHistoryEntry struct {
	UserID      uuid.UUID `json:"user_id"`
	DeviceID    uuid.UUID `json:"device_id"`
	DeviceName  string    `json:"device_name"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	Altitude    *float64  `json:"altitude,omitempty"`
	Heading     *float64  `json:"heading,omitempty"`
	Speed       *float64  `json:"speed,omitempty"`
	RecordedAt  time.Time `json:"recorded_at"`
}

// toHistoryEntry maps a LocationRecord to a LocationHistoryEntry.
func toHistoryEntry(rec repository.LocationRecord) LocationHistoryEntry {
	dn := ""
	if rec.DisplayName != nil {
		dn = *rec.DisplayName
	}
	devName := ""
	if rec.DeviceName != nil {
		devName = *rec.DeviceName
	}
	return LocationHistoryEntry{
		UserID:      rec.UserID,
		DeviceID:    rec.DeviceID,
		DeviceName:  devName,
		Username:    rec.Username,
		DisplayName: dn,
		Lat:         rec.Lat,
		Lng:         rec.Lng,
		Altitude:    rec.Altitude,
		Heading:     rec.Heading,
		Speed:       rec.Speed,
		RecordedAt:  rec.RecordedAt,
	}
}

// GetGroupHistory returns location history for a group within a time range.
// Checks that the caller is a member of the group (or is admin).
func (s *LocationService) GetGroupHistory(ctx context.Context, groupID, callerID uuid.UUID, callerIsAdmin bool, from, to time.Time) ([]LocationHistoryEntry, error) {
	// Permission check: must be admin or group member
	if !callerIsAdmin {
		if _, err := s.groupRepo.GetMember(ctx, groupID, callerID); err != nil {
			return nil, fmt.Errorf("you are not a member of this group: %w", err)
		}
	}

	records, err := s.locationRepo.GetGroupHistory(ctx, groupID, from, to)
	if err != nil {
		return nil, err
	}

	entries := make([]LocationHistoryEntry, len(records))
	for i, rec := range records {
		entries[i] = toHistoryEntry(rec)
	}

	return entries, nil
}

// GetMyHistory returns location history for the calling user within a time range.
// If deviceID is non-nil, results are filtered to that specific device.
func (s *LocationService) GetMyHistory(ctx context.Context, callerID uuid.UUID, from, to time.Time, deviceID *uuid.UUID) ([]LocationHistoryEntry, error) {
	records, err := s.locationRepo.GetUserHistory(ctx, callerID, from, to, deviceID)
	if err != nil {
		return nil, err
	}

	entries := make([]LocationHistoryEntry, len(records))
	for i, rec := range records {
		entries[i] = toHistoryEntry(rec)
	}

	return entries, nil
}

// GetVisibleHistory returns location history for all users visible to the caller.
// Admins see all users; non-admins see users who share a group with them.
func (s *LocationService) GetVisibleHistory(ctx context.Context, callerID uuid.UUID, callerIsAdmin bool, from, to time.Time) ([]LocationHistoryEntry, error) {
	var records []repository.LocationRecord
	var err error

	if callerIsAdmin {
		records, err = s.locationRepo.GetAllHistory(ctx, from, to)
	} else {
		records, err = s.locationRepo.GetVisibleHistory(ctx, callerID, from, to)
	}
	if err != nil {
		return nil, err
	}

	entries := make([]LocationHistoryEntry, len(records))
	for i, rec := range records {
		entries[i] = toHistoryEntry(rec)
	}

	return entries, nil
}

// GetUserHistory returns location history for a specific user.
// Admins can query any user; non-admins can only query users who share a group.
// If deviceID is non-nil, results are filtered to that specific device.
func (s *LocationService) GetUserHistory(ctx context.Context, targetUserID, callerID uuid.UUID, callerIsAdmin bool, from, to time.Time, deviceID *uuid.UUID) ([]LocationHistoryEntry, error) {
	// Permission check: must be admin, the user themselves, or share a group
	if !callerIsAdmin && targetUserID != callerID {
		shared, err := s.locationRepo.UsersShareGroup(ctx, callerID, targetUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to check group membership: %w", err)
		}
		if !shared {
			return nil, fmt.Errorf("you do not share a group with this user")
		}
	}

	records, err := s.locationRepo.GetUserHistory(ctx, targetUserID, from, to, deviceID)
	if err != nil {
		return nil, err
	}

	entries := make([]LocationHistoryEntry, len(records))
	for i, rec := range records {
		entries[i] = toHistoryEntry(rec)
	}

	return entries, nil
}

// LatestLocationEntry is a single device's latest known position.
type LatestLocationEntry struct {
	UserID      uuid.UUID `json:"user_id"`
	DeviceID    uuid.UUID `json:"device_id"`
	DeviceName  string    `json:"device_name"`
	IsPrimary   bool      `json:"is_primary"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	Altitude    *float64  `json:"altitude,omitempty"`
	Heading     *float64  `json:"heading,omitempty"`
	Speed       *float64  `json:"speed,omitempty"`
	RecordedAt  time.Time `json:"recorded_at"`
}

// GetAllLatest returns the latest location for every user (admin only).
func (s *LocationService) GetAllLatest(ctx context.Context) ([]LatestLocationEntry, error) {
	records, err := s.locationRepo.GetAllLatest(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]LatestLocationEntry, len(records))
	for i, rec := range records {
		dn := ""
		if rec.DisplayName != nil {
			dn = *rec.DisplayName
		}
		devName := ""
		if rec.DeviceName != nil {
			devName = *rec.DeviceName
		}
		entries[i] = LatestLocationEntry{
			UserID:      rec.UserID,
			DeviceID:    rec.DeviceID,
			DeviceName:  devName,
			IsPrimary:   rec.IsPrimary,
			Username:    rec.Username,
			DisplayName: dn,
			Lat:         rec.Lat,
			Lng:         rec.Lng,
			Altitude:    rec.Altitude,
			Heading:     rec.Heading,
			Speed:       rec.Speed,
			RecordedAt:  rec.RecordedAt,
		}
	}

	return entries, nil
}
