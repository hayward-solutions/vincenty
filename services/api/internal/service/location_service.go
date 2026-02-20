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
	locationRepo *repository.LocationRepository
	groupRepo    *repository.GroupRepository
	ps           pubsub.PubSub
	throttle     time.Duration

	// Per-device throttle tracking
	lastUpdate sync.Map // map[uuid.UUID]time.Time
}

// NewLocationService creates a new LocationService.
func NewLocationService(
	locationRepo *repository.LocationRepository,
	groupRepo *repository.GroupRepository,
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
	username, displayName string,
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
	broadcast := LocationBroadcast{
		UserID:      userID,
		Username:    username,
		DisplayName: displayName,
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

	return true, nil
}

// GetGroupSnapshot returns the latest position per user in a group.
func (s *LocationService) GetGroupSnapshot(ctx context.Context, groupID uuid.UUID) ([]repository.LocationRecord, error) {
	return s.locationRepo.GetLatestByGroup(ctx, groupID)
}

// LocationHistoryEntry is a single point in a location track.
type LocationHistoryEntry struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	Altitude    *float64  `json:"altitude,omitempty"`
	Heading     *float64  `json:"heading,omitempty"`
	Speed       *float64  `json:"speed,omitempty"`
	RecordedAt  time.Time `json:"recorded_at"`
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
		dn := ""
		if rec.DisplayName != nil {
			dn = *rec.DisplayName
		}
		entries[i] = LocationHistoryEntry{
			UserID:      rec.UserID,
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

// GetMyHistory returns location history for the calling user within a time range.
func (s *LocationService) GetMyHistory(ctx context.Context, callerID uuid.UUID, from, to time.Time) ([]LocationHistoryEntry, error) {
	records, err := s.locationRepo.GetUserHistory(ctx, callerID, from, to)
	if err != nil {
		return nil, err
	}

	entries := make([]LocationHistoryEntry, len(records))
	for i, rec := range records {
		dn := ""
		if rec.DisplayName != nil {
			dn = *rec.DisplayName
		}
		entries[i] = LocationHistoryEntry{
			UserID:      rec.UserID,
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

// LatestLocationEntry is a single user's latest known position.
type LatestLocationEntry struct {
	UserID      uuid.UUID `json:"user_id"`
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
		entries[i] = LatestLocationEntry{
			UserID:      rec.UserID,
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
