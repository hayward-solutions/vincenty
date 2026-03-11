package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/livekit"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// RecordingService handles recording business logic.
type RecordingService struct {
	recRepo    repository.RecordingRepo
	roomRepo   repository.MediaRoomRepo
	streamRepo repository.StreamRepo
	groupRepo  repository.GroupRepo
	permSvc    *PermissionPolicyService
	lk         *livekit.Client
	s3Cfg      livekit.S3Config
}

// NewRecordingService creates a new RecordingService.
func NewRecordingService(
	recRepo repository.RecordingRepo,
	roomRepo repository.MediaRoomRepo,
	streamRepo repository.StreamRepo,
	groupRepo repository.GroupRepo,
	permSvc *PermissionPolicyService,
	lk *livekit.Client,
	s3Cfg livekit.S3Config,
) *RecordingService {
	return &RecordingService{
		recRepo:    recRepo,
		roomRepo:   roomRepo,
		streamRepo: streamRepo,
		groupRepo:  groupRepo,
		permSvc:    permSvc,
		lk:         lk,
		s3Cfg:      s3Cfg,
	}
}

// StartStreamRecording starts recording an active stream's media room.
func (s *RecordingService) StartStreamRecording(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) (*model.Recording, error) {
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	if !stream.IsActive || stream.LiveKitRoom == nil {
		return nil, model.ErrValidation("stream is not active")
	}

	// Check permission
	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, stream.GroupID, callerID)
		if err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
		if err := s.permSvc.RequireCommunication(ctx, model.ActionRecordStream, member, callerIsAdmin); err != nil {
			return nil, err
		}
	}

	// Look up the media room for this stream
	room, err := s.roomRepo.GetByLiveKitRoom(ctx, *stream.LiveKitRoom)
	if err != nil {
		return nil, err
	}

	// Start the LiveKit egress
	info, err := s.lk.StartRoomRecording(ctx, room.LiveKitRoom, s.s3Cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to start recording: %w", err)
	}

	rec := &model.Recording{
		RoomID:   &room.ID,
		StreamID: &stream.ID,
		EgressID: info.EgressId,
		FileType: "mp4",
		Status:   model.RecordingStatusRecording,
	}

	if err := s.recRepo.Create(ctx, rec); err != nil {
		// Try to stop the egress if we can't record the DB entry
		_, _ = s.lk.StopRecording(ctx, info.EgressId)
		return nil, err
	}

	slog.Info("started recording", "recording_id", rec.ID, "stream_id", streamID, "egress_id", info.EgressId)
	return rec, nil
}

// StopRecording stops an active recording.
func (s *RecordingService) StopRecording(ctx context.Context, recordingID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) (*model.Recording, error) {
	rec, err := s.recRepo.GetByID(ctx, recordingID)
	if err != nil {
		return nil, err
	}

	if rec.Status != model.RecordingStatusRecording {
		return nil, model.ErrValidation("recording is not active")
	}

	// Check permission via the stream's group
	if rec.StreamID != nil {
		stream, err := s.streamRepo.GetByID(ctx, *rec.StreamID)
		if err == nil && !callerIsAdmin {
			member, err := s.groupRepo.GetMember(ctx, stream.GroupID, callerID)
			if err != nil {
				return nil, model.ErrForbidden("you are not a member of this group")
			}
			if err := s.permSvc.RequireCommunication(ctx, model.ActionRecordStream, member, callerIsAdmin); err != nil {
				return nil, err
			}
		}
	}

	info, err := s.lk.StopRecording(ctx, rec.EgressID)
	if err != nil {
		slog.Error("failed to stop egress", "error", err, "egress_id", rec.EgressID)
		rec.Status = model.RecordingStatusFailed
		_ = s.recRepo.Update(ctx, rec)
		return rec, fmt.Errorf("failed to stop recording: %w", err)
	}

	rec.Status = model.RecordingStatusProcessing
	if len(info.GetFileResults()) > 0 {
		path := info.GetFileResults()[0].Filename
		rec.StoragePath = &path
	}
	if err := s.recRepo.Update(ctx, rec); err != nil {
		return nil, err
	}

	slog.Info("stopped recording", "recording_id", rec.ID, "egress_id", rec.EgressID)
	return rec, nil
}

// CompleteRecording is called by the LiveKit webhook when an egress finishes.
func (s *RecordingService) CompleteRecording(ctx context.Context, egressID string, storagePath string, durationSecs int, fileSizeBytes int64) error {
	rec, err := s.recRepo.GetByEgressID(ctx, egressID)
	if err != nil {
		return err
	}

	rec.Status = model.RecordingStatusComplete
	rec.StoragePath = &storagePath
	rec.DurationSecs = &durationSecs
	rec.FileSizeBytes = &fileSizeBytes
	now := rec.StartedAt.Add(0)
	rec.EndedAt = &now

	return s.recRepo.Update(ctx, rec)
}

// GetByID retrieves a recording.
func (s *RecordingService) GetByID(ctx context.Context, recordingID uuid.UUID) (*model.Recording, error) {
	return s.recRepo.GetByID(ctx, recordingID)
}

// ListByStream retrieves all recordings for a stream.
func (s *RecordingService) ListByStream(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) ([]model.Recording, error) {
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	if !callerIsAdmin {
		if _, err := s.groupRepo.GetMember(ctx, stream.GroupID, callerID); err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
	}

	return s.recRepo.ListByStreamID(ctx, streamID)
}
