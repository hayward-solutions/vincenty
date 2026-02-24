package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/pubsub"
	"github.com/sitaware/api/internal/repository"
	"github.com/sitaware/api/internal/storage"
)

// ---------------------------------------------------------------------------
// Request / Response DTOs
// ---------------------------------------------------------------------------

// CreateStreamRequest contains the fields needed to start a browser stream.
type CreateStreamRequest struct {
	Title    string      `json:"title"`
	GroupIDs []uuid.UUID `json:"group_ids"`
}

// ShareStreamRequest contains additional groups to share a stream with.
type ShareStreamRequest struct {
	GroupIDs []uuid.UUID `json:"group_ids"`
}

// MediaAuthRequest is sent by MediaMTX to validate publish/read access.
type MediaAuthRequest struct {
	IP       string `json:"ip"`
	User     string `json:"user"`
	Password string `json:"password"`
	Path     string `json:"path"`
	Protocol string `json:"protocol"`
	Action   string `json:"action"`
	Query    string `json:"query"`
}

// RecordingCompleteRequest is sent by MediaMTX when a recording segment finishes.
type RecordingCompleteRequest struct {
	Path     string `json:"path"`
	FilePath string `json:"filePath"`
}

// CreateStreamKeyRequest contains the fields needed to create a stream key.
type CreateStreamKeyRequest struct {
	Label    string      `json:"label"`
	GroupIDs []uuid.UUID `json:"group_ids"`
}

// UpdateStreamKeyRequest contains the fields that can be updated on a stream key.
type UpdateStreamKeyRequest struct {
	Label    *string      `json:"label,omitempty"`
	IsActive *bool        `json:"is_active,omitempty"`
	GroupIDs *[]uuid.UUID `json:"group_ids,omitempty"`
}

// ---------------------------------------------------------------------------
// StreamService
// ---------------------------------------------------------------------------

// StreamService handles stream business logic.
type StreamService struct {
	streamRepo    *repository.StreamRepository
	streamKeyRepo *repository.StreamKeyRepository
	groupRepo     *repository.GroupRepository
	storageSvc    *storage.StorageService
	ps            pubsub.PubSub
	mediaMTXURL   string // base URL for WHIP/WHEP (e.g. "http://mediamtx:8889")
}

// NewStreamService creates a new StreamService.
func NewStreamService(
	streamRepo *repository.StreamRepository,
	streamKeyRepo *repository.StreamKeyRepository,
	groupRepo *repository.GroupRepository,
	storageSvc *storage.StorageService,
	ps pubsub.PubSub,
	mediaMTXURL string,
) *StreamService {
	return &StreamService{
		streamRepo:    streamRepo,
		streamKeyRepo: streamKeyRepo,
		groupRepo:     groupRepo,
		storageSvc:    storageSvc,
		ps:            ps,
		mediaMTXURL:   mediaMTXURL,
	}
}

// Create starts a new browser-based stream.
func (s *StreamService) Create(ctx context.Context, callerID uuid.UUID, req CreateStreamRequest) (*model.StreamWithDetails, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, model.ErrValidation("title is required")
	}

	if len(req.GroupIDs) == 0 {
		return nil, model.ErrValidation("at least one group_id is required")
	}

	// Validate caller is a member of all requested groups
	for _, gid := range req.GroupIDs {
		_, err := s.groupRepo.GetMember(ctx, gid, callerID)
		if err != nil {
			return nil, model.ErrForbidden(fmt.Sprintf("you are not a member of group %s", gid))
		}
	}

	streamID := uuid.New()
	mediaPath := streamID.String()

	stream := &model.Stream{
		ID:            streamID,
		Title:         title,
		BroadcasterID: &callerID,
		SourceType:    "browser",
		Status:        "live",
		MediaPath:     mediaPath,
	}

	if err := s.streamRepo.Create(ctx, stream, req.GroupIDs); err != nil {
		return nil, fmt.Errorf("create stream: %w", err)
	}

	// Re-fetch to get joined data
	swd, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	// Broadcast stream_started to all shared groups
	s.broadcastStreamEvent(ctx, swd, "stream_started")

	return swd, nil
}

// Get retrieves a stream by ID. Caller must be a member of at least one shared group.
func (s *StreamService) Get(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) (*model.StreamWithDetails, error) {
	swd, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	if !callerIsAdmin {
		if err := s.checkStreamAccess(ctx, swd, callerID); err != nil {
			return nil, err
		}
	}

	return swd, nil
}

// List returns streams visible to the caller via their group memberships.
func (s *StreamService) List(ctx context.Context, callerID uuid.UUID, status string) ([]model.StreamWithDetails, error) {
	groups, _, err := s.groupRepo.ListByUserID(ctx, callerID)
	if err != nil {
		return nil, fmt.Errorf("list user groups: %w", err)
	}

	groupIDs := make([]uuid.UUID, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID
	}

	if len(groupIDs) == 0 {
		return []model.StreamWithDetails{}, nil
	}

	streams, err := s.streamRepo.ListByGroupIDs(ctx, groupIDs, status)
	if err != nil {
		return nil, fmt.Errorf("list streams: %w", err)
	}

	if streams == nil {
		streams = []model.StreamWithDetails{}
	}

	return streams, nil
}

// Share adds groups to a stream. Only the broadcaster or an admin can share.
func (s *StreamService) Share(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool, req ShareStreamRequest) error {
	swd, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}

	if !callerIsAdmin && (swd.BroadcasterID == nil || *swd.BroadcasterID != callerID) {
		return model.ErrForbidden("only the broadcaster or an admin can share this stream")
	}

	for _, gid := range req.GroupIDs {
		if err := s.streamRepo.AddGroup(ctx, streamID, gid); err != nil {
			return fmt.Errorf("add stream group: %w", err)
		}
	}

	// Re-fetch and broadcast to new groups
	updated, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}
	s.broadcastStreamEvent(ctx, updated, "stream_started")

	return nil
}

// Unshare removes a group from a stream.
func (s *StreamService) Unshare(ctx context.Context, streamID, groupID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) error {
	swd, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}

	if !callerIsAdmin && (swd.BroadcasterID == nil || *swd.BroadcasterID != callerID) {
		return model.ErrForbidden("only the broadcaster or an admin can unshare this stream")
	}

	return s.streamRepo.RemoveGroup(ctx, streamID, groupID)
}

// End stops a live stream.
func (s *StreamService) End(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) error {
	swd, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}

	if swd.Status != "live" {
		return model.ErrValidation("stream is not live")
	}

	if !callerIsAdmin && (swd.BroadcasterID == nil || *swd.BroadcasterID != callerID) {
		return model.ErrForbidden("only the broadcaster or an admin can end this stream")
	}

	now := time.Now()
	if err := s.streamRepo.UpdateStatus(ctx, streamID, "ended", &now); err != nil {
		return fmt.Errorf("end stream: %w", err)
	}

	// Broadcast stream_ended
	s.broadcastStreamEnded(ctx, swd)

	return nil
}

// Delete removes a stream record.
func (s *StreamService) Delete(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) error {
	swd, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}

	if !callerIsAdmin && (swd.BroadcasterID == nil || *swd.BroadcasterID != callerID) {
		return model.ErrForbidden("only the broadcaster or an admin can delete this stream")
	}

	return s.streamRepo.Delete(ctx, streamID)
}

// RecordLocation stores a GPS telemetry point for a stream.
func (s *StreamService) RecordLocation(ctx context.Context, streamID uuid.UUID, lat, lng float64, altitude, heading, speed *float64) error {
	loc := &model.StreamLocation{
		StreamID:   streamID,
		Lat:        lat,
		Lng:        lng,
		Altitude:   altitude,
		Heading:    heading,
		Speed:      speed,
		RecordedAt: time.Now(),
	}
	return s.streamRepo.CreateLocation(ctx, loc)
}

// GetLocations returns the GPS telemetry for a stream.
func (s *StreamService) GetLocations(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) ([]model.StreamLocationResponse, error) {
	swd, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	if !callerIsAdmin {
		if err := s.checkStreamAccess(ctx, swd, callerID); err != nil {
			return nil, err
		}
	}

	locations, err := s.streamRepo.GetLocations(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("get stream locations: %w", err)
	}

	resp := make([]model.StreamLocationResponse, len(locations))
	for i, loc := range locations {
		resp[i] = model.StreamLocationResponse{
			Lat:        loc.Lat,
			Lng:        loc.Lng,
			Altitude:   loc.Altitude,
			Heading:    loc.Heading,
			Speed:      loc.Speed,
			RecordedAt: loc.RecordedAt,
		}
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// MediaMTX Auth Hook
// ---------------------------------------------------------------------------

// AuthenticateMedia validates publish/read requests from MediaMTX.
// Returns nil to allow, error to deny.
func (s *StreamService) AuthenticateMedia(ctx context.Context, req MediaAuthRequest) error {
	path := strings.TrimPrefix(req.Path, "/")

	switch req.Action {
	case "publish":
		return s.authPublish(ctx, path, req)
	case "read":
		return s.authRead(ctx, path, req)
	default:
		return model.ErrForbidden("unknown action")
	}
}

// authPublish validates that the publisher is authorized.
func (s *StreamService) authPublish(ctx context.Context, path string, req MediaAuthRequest) error {
	switch req.Protocol {
	case "webrtc":
		// Browser publish via WHIP — JWT token in query string
		token := extractQueryParam(req.Query, "token")
		if token == "" {
			return model.ErrForbidden("missing token")
		}
		// The stream must already exist with this media_path and status=live
		_, err := s.streamRepo.GetByMediaPath(ctx, path)
		if err != nil {
			return model.ErrForbidden("stream not found")
		}
		// Token validation is delegated to the caller (handler validates JWT)
		return nil

	case "rtsp", "rtmp":
		// Hardware device — password is the stream key
		if req.Password == "" {
			return model.ErrForbidden("missing stream key")
		}
		keyHash := hashStreamKey(req.Password)
		sk, err := s.streamKeyRepo.GetByKeyHash(ctx, keyHash)
		if err != nil {
			return model.ErrForbidden("invalid stream key")
		}

		// Auto-create a stream record for this hardware device
		streamID := uuid.New()
		stream := &model.Stream{
			ID:          streamID,
			Title:       sk.Label,
			StreamKeyID: &sk.ID,
			SourceType:  req.Protocol,
			Status:      "live",
			MediaPath:   path,
		}

		groupIDs := sk.GroupIDs
		if groupIDs == nil {
			groupIDs = []uuid.UUID{}
		}

		if err := s.streamRepo.Create(ctx, stream, groupIDs); err != nil {
			return fmt.Errorf("auto-create stream: %w", err)
		}

		swd, err := s.streamRepo.GetByID(ctx, streamID)
		if err == nil {
			s.broadcastStreamEvent(ctx, swd, "stream_started")
		}

		return nil

	default:
		return model.ErrForbidden("unsupported protocol")
	}
}

// authRead validates that the viewer is authorized.
func (s *StreamService) authRead(ctx context.Context, path string, req MediaAuthRequest) error {
	// Viewer must provide a JWT token in the query string
	token := extractQueryParam(req.Query, "token")
	if token == "" {
		return model.ErrForbidden("missing token")
	}

	// The stream must exist
	_, err := s.streamRepo.GetByMediaPath(ctx, path)
	if err != nil {
		return model.ErrForbidden("stream not found")
	}

	// Full JWT validation + group membership check is handled by the handler
	// which calls this service. We return nil here to indicate the stream exists.
	return nil
}

// HandleRecordingComplete processes a MediaMTX recording segment callback.
func (s *StreamService) HandleRecordingComplete(ctx context.Context, req RecordingCompleteRequest) error {
	path := strings.TrimPrefix(req.Path, "/")

	swd, err := s.streamRepo.GetByMediaPath(ctx, path)
	if err != nil {
		slog.Warn("recording complete for unknown stream", "path", path)
		return nil // Don't error — MediaMTX may send for paths we don't track
	}

	// Store the recording URL as the S3 key pattern
	s3Key := fmt.Sprintf("streams/%s/recording.mp4", swd.ID)
	if err := s.streamRepo.SetRecordingURL(ctx, swd.ID, s3Key); err != nil {
		return fmt.Errorf("set recording URL: %w", err)
	}

	slog.Info("recording complete",
		"stream_id", swd.ID,
		"path", path,
		"file_path", req.FilePath,
		"s3_key", s3Key,
	)

	return nil
}

// GetMediaMTXURL returns the base MediaMTX URL for constructing WHIP/WHEP endpoints.
func (s *StreamService) GetMediaMTXURL() string {
	return s.mediaMTXURL
}

// ---------------------------------------------------------------------------
// Stream Key Management
// ---------------------------------------------------------------------------

// CreateStreamKey generates a new stream key for hardware devices.
func (s *StreamService) CreateStreamKey(ctx context.Context, callerID uuid.UUID, req CreateStreamKeyRequest) (*model.StreamKeyResponse, error) {
	label := strings.TrimSpace(req.Label)
	if label == "" {
		return nil, model.ErrValidation("label is required")
	}

	// Generate a random stream key (32 bytes = 64 hex chars)
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("generate stream key: %w", err)
	}
	plaintext := hex.EncodeToString(keyBytes)
	keyHash := hashStreamKey(plaintext)

	sk := &model.StreamKey{
		Label:     label,
		KeyHash:   keyHash,
		CreatedBy: callerID,
		IsActive:  true,
	}

	groupIDs := req.GroupIDs
	if groupIDs == nil {
		groupIDs = []uuid.UUID{}
	}

	if err := s.streamKeyRepo.Create(ctx, sk, groupIDs); err != nil {
		return nil, fmt.Errorf("create stream key: %w", err)
	}

	kwg, err := s.streamKeyRepo.GetByID(ctx, sk.ID)
	if err != nil {
		return nil, err
	}

	resp := kwg.ToResponse()
	resp.Key = plaintext // Only returned on creation
	return &resp, nil
}

// ListStreamKeys returns all stream keys.
func (s *StreamService) ListStreamKeys(ctx context.Context) ([]model.StreamKeyResponse, error) {
	keys, err := s.streamKeyRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list stream keys: %w", err)
	}

	resp := make([]model.StreamKeyResponse, len(keys))
	for i := range keys {
		resp[i] = keys[i].ToResponse()
	}
	return resp, nil
}

// UpdateStreamKey modifies a stream key.
func (s *StreamService) UpdateStreamKey(ctx context.Context, keyID uuid.UUID, req UpdateStreamKeyRequest) (*model.StreamKeyResponse, error) {
	existing, err := s.streamKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	label := existing.Label
	isActive := existing.IsActive

	if req.Label != nil {
		label = strings.TrimSpace(*req.Label)
		if label == "" {
			return nil, model.ErrValidation("label cannot be empty")
		}
	}
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	if err := s.streamKeyRepo.Update(ctx, keyID, label, isActive); err != nil {
		return nil, fmt.Errorf("update stream key: %w", err)
	}

	if req.GroupIDs != nil {
		if err := s.streamKeyRepo.SetGroups(ctx, keyID, *req.GroupIDs); err != nil {
			return nil, fmt.Errorf("set stream key groups: %w", err)
		}
	}

	updated, err := s.streamKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	resp := updated.ToResponse()
	return &resp, nil
}

// DeleteStreamKey removes a stream key.
func (s *StreamService) DeleteStreamKey(ctx context.Context, keyID uuid.UUID) error {
	return s.streamKeyRepo.Delete(ctx, keyID)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// checkStreamAccess verifies the caller is a member of at least one shared group.
func (s *StreamService) checkStreamAccess(ctx context.Context, swd *model.StreamWithDetails, callerID uuid.UUID) error {
	// Broadcaster always has access
	if swd.BroadcasterID != nil && *swd.BroadcasterID == callerID {
		return nil
	}

	for _, gid := range swd.Groups {
		_, err := s.groupRepo.GetMember(ctx, gid, callerID)
		if err == nil {
			return nil // Member of at least one shared group
		}
	}
	return model.ErrForbidden("you do not have access to this stream")
}

// broadcastStreamEvent publishes a stream event to all shared groups.
func (s *StreamService) broadcastStreamEvent(ctx context.Context, swd *model.StreamWithDetails, eventType string) {
	data, err := json.Marshal(swd.ToResponse())
	if err != nil {
		slog.Error("failed to marshal stream for broadcast", "error", err)
		return
	}

	for _, gid := range swd.Groups {
		channel := fmt.Sprintf("group:%s:streams", gid)
		if err := s.ps.Publish(ctx, channel, data); err != nil {
			slog.Error("failed to publish stream event", "error", err, "channel", channel, "event", eventType)
		} else {
			slog.Debug("published stream event", "channel", channel, "event", eventType, "stream_id", swd.ID)
		}
	}
}

// broadcastStreamEnded publishes a stream_ended event.
func (s *StreamService) broadcastStreamEnded(ctx context.Context, swd *model.StreamWithDetails) {
	payload := struct {
		StreamID uuid.UUID `json:"stream_id"`
	}{StreamID: swd.ID}

	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal stream_ended", "error", err)
		return
	}

	for _, gid := range swd.Groups {
		channel := fmt.Sprintf("group:%s:streams", gid)
		if err := s.ps.Publish(ctx, channel, data); err != nil {
			slog.Error("failed to publish stream_ended", "error", err, "channel", channel)
		}
	}
}

// BroadcastStreamLocation publishes a stream location update to shared groups.
func (s *StreamService) BroadcastStreamLocation(ctx context.Context, streamID uuid.UUID, lat, lng float64, altitude, heading *float64) {
	swd, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		slog.Error("failed to get stream for location broadcast", "error", err, "stream_id", streamID)
		return
	}

	payload := struct {
		StreamID uuid.UUID `json:"stream_id"`
		Lat      float64   `json:"lat"`
		Lng      float64   `json:"lng"`
		Altitude *float64  `json:"altitude,omitempty"`
		Heading  *float64  `json:"heading,omitempty"`
	}{
		StreamID: streamID,
		Lat:      lat,
		Lng:      lng,
		Altitude: altitude,
		Heading:  heading,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal stream location", "error", err)
		return
	}

	for _, gid := range swd.Groups {
		channel := fmt.Sprintf("group:%s:stream_locations", gid)
		if err := s.ps.Publish(ctx, channel, data); err != nil {
			slog.Error("failed to publish stream location", "error", err, "channel", channel)
		}
	}
}

// hashStreamKey returns the SHA-256 hex digest of a stream key.
func hashStreamKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// extractQueryParam extracts a value from a URL query string.
func extractQueryParam(queryStr, key string) string {
	values, err := url.ParseQuery(queryStr)
	if err != nil {
		return ""
	}
	return values.Get(key)
}
