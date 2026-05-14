package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/garmin"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/repository"
)

// GarminInReachService handles feed CRUD, KML polling, and location bridging.
type GarminInReachService struct {
	feedRepo    repository.GarminInReachRepo
	deviceRepo  repository.DeviceRepo
	userRepo    repository.UserRepo
	groupRepo   repository.GroupRepo
	locationSvc *LocationService
	httpClient  *http.Client
}

// NewGarminInReachService creates a new GarminInReachService.
func NewGarminInReachService(
	feedRepo repository.GarminInReachRepo,
	deviceRepo repository.DeviceRepo,
	userRepo repository.UserRepo,
	groupRepo repository.GroupRepo,
	locationSvc *LocationService,
) *GarminInReachService {
	return &GarminInReachService{
		feedRepo:    feedRepo,
		deviceRepo:  deviceRepo,
		userRepo:    userRepo,
		groupRepo:   groupRepo,
		locationSvc: locationSvc,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Create creates a new InReach feed and auto-creates a "garmin_inreach" device
// for the target user if needed.
func (s *GarminInReachService) Create(ctx context.Context, req model.CreateGarminInReachFeedRequest) (*model.GarminInReachFeed, error) {
	// Verify user exists
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Create a dedicated device for this InReach feed
	pollInterval, _ := time.ParseDuration(req.PollInterval)
	deviceName := fmt.Sprintf("InReach (%s)", req.MapShareID)

	device := &model.Device{
		ID:         uuid.New(),
		UserID:     user.ID,
		Name:       deviceName,
		DeviceType: "garmin_inreach",
	}
	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}

	feed := &model.GarminInReachFeed{
		UserID:       req.UserID,
		DeviceID:     device.ID,
		MapShareID:   req.MapShareID,
		FeedPassword: req.FeedPassword,
		PollInterval: pollInterval,
		Enabled:      true,
	}

	if err := s.feedRepo.Create(ctx, feed); err != nil {
		return nil, err
	}

	slog.Info("garmin inreach: feed created",
		"feed_id", feed.ID,
		"user_id", feed.UserID,
		"device_id", feed.DeviceID,
		"mapshare_id", feed.MapShareID,
	)

	return feed, nil
}

// Get returns a feed by ID.
func (s *GarminInReachService) Get(ctx context.Context, id uuid.UUID) (*model.GarminInReachFeed, error) {
	return s.feedRepo.GetByID(ctx, id)
}

// List returns all configured feeds.
func (s *GarminInReachService) List(ctx context.Context) ([]model.GarminInReachFeed, error) {
	return s.feedRepo.List(ctx)
}

// Update updates a feed's configurable fields.
func (s *GarminInReachService) Update(ctx context.Context, id uuid.UUID, req model.UpdateGarminInReachFeedRequest) (*model.GarminInReachFeed, error) {
	feed, err := s.feedRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.FeedPassword != nil {
		feed.FeedPassword = req.FeedPassword
	}
	if req.PollInterval != nil {
		d, _ := time.ParseDuration(*req.PollInterval)
		feed.PollInterval = d
	}
	if req.Enabled != nil {
		feed.Enabled = *req.Enabled
	}

	if err := s.feedRepo.Update(ctx, feed); err != nil {
		return nil, err
	}
	return feed, nil
}

// Delete removes a feed and its associated device.
func (s *GarminInReachService) Delete(ctx context.Context, id uuid.UUID) error {
	feed, err := s.feedRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.feedRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Clean up the dedicated device
	if delErr := s.deviceRepo.Delete(ctx, feed.DeviceID); delErr != nil {
		slog.Warn("garmin inreach: failed to delete device after feed removal",
			"device_id", feed.DeviceID,
			"error", delErr,
		)
	}

	return nil
}

// PollResult summarizes the outcome of polling a single feed.
type PollResult struct {
	FeedID  uuid.UUID `json:"feed_id"`
	Points  int       `json:"points"`
	Bridged int       `json:"bridged"`
	Error   string    `json:"error,omitempty"`
}

// PollAll fetches KML data for all enabled feeds that are due for polling
// and bridges new track points into the location system.
func (s *GarminInReachService) PollAll(ctx context.Context) []PollResult {
	feeds, err := s.feedRepo.ListEnabled(ctx)
	if err != nil {
		slog.Error("garmin inreach: failed to list enabled feeds", "error", err)
		return nil
	}

	if len(feeds) == 0 {
		return nil
	}

	slog.Debug("garmin inreach: polling feeds", "count", len(feeds))

	var results []PollResult
	for _, feed := range feeds {
		result := s.pollFeed(ctx, feed)
		results = append(results, result)
	}
	return results
}

// pollFeed fetches and processes a single feed.
func (s *GarminInReachService) pollFeed(ctx context.Context, feed model.GarminInReachFeed) PollResult {
	result := PollResult{FeedID: feed.ID}

	points, err := s.fetchKML(ctx, feed)
	if err != nil {
		result.Error = err.Error()
		_ = s.feedRepo.UpdatePollStatus(ctx, feed.ID, nil, err)
		slog.Error("garmin inreach: poll failed",
			"feed_id", feed.ID,
			"mapshare_id", feed.MapShareID,
			"error", err,
		)
		return result
	}

	result.Points = len(points)

	// Filter to only new points (after last_point_at)
	var newPoints []garmin.TrackPoint
	for _, pt := range points {
		if feed.LastPointAt == nil || pt.Timestamp.After(*feed.LastPointAt) {
			newPoints = append(newPoints, pt)
		}
	}

	// Bridge each new point to the location system
	var latestTime *time.Time
	for _, pt := range newPoints {
		if err := s.bridgePoint(ctx, feed, pt); err != nil {
			slog.Error("garmin inreach: bridge failed",
				"feed_id", feed.ID,
				"timestamp", pt.Timestamp,
				"error", err,
			)
			continue
		}
		result.Bridged++
		t := pt.Timestamp
		if latestTime == nil || t.After(*latestTime) {
			latestTime = &t
		}
	}

	_ = s.feedRepo.UpdatePollStatus(ctx, feed.ID, latestTime, nil)

	slog.Info("garmin inreach: poll complete",
		"feed_id", feed.ID,
		"mapshare_id", feed.MapShareID,
		"total_points", len(points),
		"new_points", len(newPoints),
		"bridged", result.Bridged,
	)

	return result
}

// fetchKML downloads and parses the MapShare KML feed.
func (s *GarminInReachService) fetchKML(ctx context.Context, feed model.GarminInReachFeed) ([]garmin.TrackPoint, error) {
	feedURL := fmt.Sprintf("https://share.garmin.com/Feed/Share/%s", url.PathEscape(feed.MapShareID))

	// If we have a last_point_at, request only data since then
	if feed.LastPointAt != nil {
		params := url.Values{}
		params.Set("d1", feed.LastPointAt.UTC().Format("2006-01-02T15:04:05Z"))
		feedURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.google-earth.kml+xml")

	// If MapShare requires a password, send it as basic auth
	if feed.FeedPassword != nil && *feed.FeedPassword != "" {
		req.SetBasicAuth("", *feed.FeedPassword)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("feed returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	return garmin.ParseKML(resp.Body)
}

// bridgePoint converts a KML track point into a location update.
func (s *GarminInReachService) bridgePoint(ctx context.Context, feed model.GarminInReachFeed, pt garmin.TrackPoint) error {
	// Resolve user info
	user, err := s.userRepo.GetByID(ctx, feed.UserID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}
	displayName := ""
	if user.DisplayName != nil {
		displayName = *user.DisplayName
	}

	// Get device info
	device, err := s.deviceRepo.GetByID(ctx, feed.DeviceID)
	if err != nil {
		return fmt.Errorf("resolve device: %w", err)
	}

	// Get user's groups for broadcasting
	groups, _, err := s.groupRepo.ListByUserID(ctx, feed.UserID)
	if err != nil {
		return fmt.Errorf("list user groups: %w", err)
	}
	groupIDs := make([]uuid.UUID, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID
	}

	_, err = s.locationSvc.Update(
		ctx,
		feed.UserID, feed.DeviceID,
		user.Username, displayName, device.Name,
		device.IsPrimary,
		pt.Lat, pt.Lng,
		pt.Altitude, pt.Course, pt.Speed, nil,
		groupIDs,
	)
	return err
}

// RunPoller starts a background goroutine that periodically polls all
// enabled feeds. It blocks until the context is cancelled.
func (s *GarminInReachService) RunPoller(ctx context.Context, tick time.Duration) {
	slog.Info("garmin inreach: poller started", "tick", tick)
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.PollAll(ctx)
		case <-ctx.Done():
			slog.Info("garmin inreach: poller stopped")
			return
		}
	}
}

// IngestWebhook processes a Garmin Explore outbound webhook POST.
// The body is KML containing track points. The mapShareID identifies
// which feed configuration to use for user/device mapping.
func (s *GarminInReachService) IngestWebhook(ctx context.Context, mapShareID string, body io.Reader) (*PollResult, error) {
	// Look up which feed this MapShare ID belongs to
	feeds, err := s.feedRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	var feed *model.GarminInReachFeed
	for i := range feeds {
		if feeds[i].MapShareID == mapShareID {
			feed = &feeds[i]
			break
		}
	}
	if feed == nil {
		return nil, model.ErrNotFound("garmin inreach feed")
	}

	points, err := garmin.ParseKML(body)
	if err != nil {
		return nil, model.ErrValidation(fmt.Sprintf("invalid KML: %s", err))
	}

	result := &PollResult{
		FeedID: feed.ID,
		Points: len(points),
	}

	var latestTime *time.Time
	for _, pt := range points {
		if feed.LastPointAt != nil && !pt.Timestamp.After(*feed.LastPointAt) {
			continue
		}
		if err := s.bridgePoint(ctx, *feed, pt); err != nil {
			slog.Error("garmin webhook: bridge failed",
				"feed_id", feed.ID,
				"error", err,
			)
			continue
		}
		result.Bridged++
		t := pt.Timestamp
		if latestTime == nil || t.After(*latestTime) {
			latestTime = &t
		}
	}

	_ = s.feedRepo.UpdatePollStatus(ctx, feed.ID, latestTime, nil)

	return result, nil
}
