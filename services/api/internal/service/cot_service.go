package service

import (
	"context"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/cot"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/repository"
)

// CotService handles CoT event ingestion, storage, and bridging to internal systems.
type CotService struct {
	cotRepo     repository.CotRepo
	deviceRepo  repository.DeviceRepo
	userRepo    repository.UserRepo
	groupRepo   repository.GroupRepo
	locationSvc *LocationService
}

// NewCotService creates a new CotService.
func NewCotService(
	cotRepo repository.CotRepo,
	deviceRepo repository.DeviceRepo,
	userRepo repository.UserRepo,
	groupRepo repository.GroupRepo,
	locationSvc *LocationService,
) *CotService {
	return &CotService{
		cotRepo:     cotRepo,
		deviceRepo:  deviceRepo,
		userRepo:    userRepo,
		groupRepo:   groupRepo,
		locationSvc: locationSvc,
	}
}

// IngestResult summarizes the outcome of a batch ingestion.
type IngestResult struct {
	Total   int `json:"total"`
	Stored  int `json:"stored"`
	Bridged int `json:"bridged"`
	Errors  int `json:"errors"`
}

// Ingest parses CoT XML from the reader, stores events, and bridges
// position events to the internal location system.
func (s *CotService) Ingest(ctx context.Context, body io.Reader) (*IngestResult, error) {
	events, err := cot.Parse(body)
	if err != nil {
		return nil, model.ErrValidation(err.Error())
	}

	result := &IngestResult{Total: len(events)}

	for i, evt := range events {
		bridged, err := s.processEvent(ctx, evt)
		if err != nil {
			slog.Error("cot ingest: failed to process event",
				"index", i,
				"uid", evt.UID,
				"type", evt.Type,
				"error", err,
			)
			result.Errors++
		} else {
			result.Stored++
			if bridged {
				result.Bridged++
			}
		}
	}

	return result, nil
}

// processEvent handles a single CoT event: resolve device, store, bridge.
// Returns true if the event was bridged to an internal system.
func (s *CotService) processEvent(ctx context.Context, evt cot.Event) (bool, error) {
	// Resolve CoT UID → device → user
	var userID *uuid.UUID
	var deviceID *uuid.UUID
	var username, displayName string

	device, err := s.deviceRepo.GetByDeviceUID(ctx, evt.UID)
	if err == nil && device != nil {
		deviceID = &device.ID
		userID = &device.UserID

		// Get user info for location broadcast
		username, displayName = s.resolveUserInfo(ctx, device.UserID)
	} else {
		slog.Debug("cot ingest: no device mapping for UID",
			"uid", evt.UID,
			"error", err,
		)
	}

	// Convert to domain model and store
	cotEvent := cot.ToCotEvent(evt, userID, deviceID)
	if err := s.cotRepo.Create(ctx, &cotEvent); err != nil {
		return false, err
	}

	// Bridge based on category
	bridged := false
	category := cot.Classify(evt.Type)
	switch category {
	case cot.CategoryPosition:
		bridged = s.bridgePosition(ctx, evt, userID, deviceID, username, displayName)
	case cot.CategoryGeoChat:
		// GeoChat bridging: store in cot_events (done above).
		// Full message bridge deferred — would need to resolve target group.
		slog.Debug("cot ingest: geochat event stored (message bridge deferred)",
			"uid", evt.UID,
		)
	default:
		slog.Debug("cot ingest: stored non-bridged event",
			"uid", evt.UID,
			"type", evt.Type,
			"category", category,
		)
	}

	return bridged, nil
}

// bridgePosition bridges a CoT position event to the internal location system.
// Returns true if the bridge was successful.
func (s *CotService) bridgePosition(ctx context.Context, evt cot.Event, userID, deviceID *uuid.UUID, username, displayName string) bool {
	bridge := cot.ToLocationBridge(evt, userID, deviceID)
	if bridge == nil {
		slog.Debug("cot bridge: skipping position (no device mapping)",
			"uid", evt.UID,
		)
		return false
	}

	// Get the user's group IDs for broadcasting
	groups, _, err := s.groupRepo.ListByUserID(ctx, bridge.UserID)
	if err != nil {
		slog.Error("cot bridge: failed to get user groups",
			"user_id", bridge.UserID,
			"error", err,
		)
		return false
	}

	groupIDs := make([]uuid.UUID, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID
	}

	// Resolve device name and primary status for the broadcast
	deviceName := ""
	isPrimary := false
	device, devErr := s.deviceRepo.GetByID(ctx, bridge.DeviceID)
	if devErr == nil {
		deviceName = device.Name
		isPrimary = device.IsPrimary
	}

	// Call LocationService.Update (this handles persist + broadcast)
	accepted, err := s.locationSvc.Update(
		ctx,
		bridge.UserID, bridge.DeviceID,
		username, displayName, deviceName,
		isPrimary,
		bridge.Lat, bridge.Lng,
		bridge.Altitude, bridge.Heading, bridge.Speed, bridge.Accuracy,
		groupIDs,
	)
	if err != nil {
		slog.Error("cot bridge: location update failed",
			"uid", evt.UID,
			"error", err,
		)
		return false
	}

	slog.Debug("cot bridge: position bridged",
		"uid", evt.UID,
		"accepted", accepted,
		"groups", len(groupIDs),
	)
	return true
}

// resolveUserInfo gets username and display name for a user ID.
func (s *CotService) resolveUserInfo(ctx context.Context, userID uuid.UUID) (string, string) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		slog.Error("cot: failed to resolve user info", "user_id", userID, "error", err)
		return "", ""
	}
	displayName := ""
	if user.DisplayName != nil {
		displayName = *user.DisplayName
	}
	return user.Username, displayName
}

// List returns CoT events matching the filters with pagination.
func (s *CotService) List(ctx context.Context, f model.CotEventFilters) ([]model.CotEvent, int, error) {
	return s.cotRepo.List(ctx, f)
}

// GetLatestByUID returns the most recent CoT event for a given UID.
func (s *CotService) GetLatestByUID(ctx context.Context, eventUID string) (*model.CotEvent, error) {
	return s.cotRepo.GetLatestByUID(ctx, eventUID)
}
