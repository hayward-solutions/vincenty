package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/livekit"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/pubsub"
	"github.com/sitaware/api/internal/repository"
)

// streamEvent is published for stream lifecycle changes.
type streamEvent struct {
	StreamID   uuid.UUID `json:"stream_id"`
	StreamName string    `json:"stream_name"`
	GroupID    uuid.UUID `json:"group_id"`
	EventType  string    `json:"event_type"` // "started", "stopped"
}

// StreamService handles stream business logic.
type StreamService struct {
	streamRepo repository.StreamRepo
	roomRepo   repository.MediaRoomRepo
	groupRepo  repository.GroupRepo
	userRepo   repository.UserRepo
	permSvc    *PermissionPolicyService
	lk         *livekit.Client
	lkURL      string
	ps         pubsub.PubSub
}

// NewStreamService creates a new StreamService.
func NewStreamService(
	streamRepo repository.StreamRepo,
	roomRepo repository.MediaRoomRepo,
	groupRepo repository.GroupRepo,
	userRepo repository.UserRepo,
	permSvc *PermissionPolicyService,
	lk *livekit.Client,
	lkURL string,
	ps pubsub.PubSub,
) *StreamService {
	return &StreamService{
		streamRepo: streamRepo,
		roomRepo:   roomRepo,
		groupRepo:  groupRepo,
		userRepo:   userRepo,
		permSvc:    permSvc,
		lk:         lk,
		lkURL:      lkURL,
		ps:         ps,
	}
}

// Create registers a new stream.
func (s *StreamService) Create(ctx context.Context, req *model.CreateStreamRequest, callerID uuid.UUID, callerIsAdmin bool) (*model.Stream, error) {
	groupID, _ := uuid.Parse(req.GroupID)

	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, groupID, callerID)
		if err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
		if err := s.permSvc.RequireCommunication(ctx, model.ActionStartStream, member, callerIsAdmin); err != nil {
			return nil, err
		}
	}

	var sourceURL *string
	if req.SourceURL != "" {
		sourceURL = &req.SourceURL
	}

	stream := &model.Stream{
		Name:       req.Name,
		SourceType: req.SourceType,
		SourceURL:  sourceURL,
		GroupID:    groupID,
		CreatedBy:  callerID,
	}

	if err := s.streamRepo.Create(ctx, stream); err != nil {
		return nil, err
	}

	return stream, nil
}

// GetByID retrieves a stream.
func (s *StreamService) GetByID(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) (*model.Stream, error) {
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	if !callerIsAdmin {
		if _, err := s.groupRepo.GetMember(ctx, stream.GroupID, callerID); err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
	}

	return stream, nil
}

// ListByGroup retrieves all streams for a group.
func (s *StreamService) ListByGroup(ctx context.Context, groupID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) ([]model.Stream, error) {
	if !callerIsAdmin {
		if _, err := s.groupRepo.GetMember(ctx, groupID, callerID); err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
	}
	return s.streamRepo.ListByGroupID(ctx, groupID)
}

// ListActive retrieves all active streams across the caller's groups.
func (s *StreamService) ListActive(ctx context.Context, callerID uuid.UUID) ([]model.Stream, error) {
	return s.streamRepo.ListActiveByUserGroups(ctx, callerID)
}

// Update modifies a stream.
func (s *StreamService) Update(ctx context.Context, streamID uuid.UUID, req *model.UpdateStreamRequest, callerID uuid.UUID, callerIsAdmin bool) (*model.Stream, error) {
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, stream.GroupID, callerID)
		if err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
		if err := s.permSvc.RequireCommunication(ctx, model.ActionStartStream, member, callerIsAdmin); err != nil {
			return nil, err
		}
	}

	if req.Name != nil {
		stream.Name = *req.Name
	}
	if req.SourceURL != nil {
		stream.SourceURL = req.SourceURL
	}

	if err := s.streamRepo.Update(ctx, stream); err != nil {
		return nil, err
	}

	return stream, nil
}

// Delete removes a stream and cleans up LiveKit resources.
func (s *StreamService) Delete(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) error {
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}

	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, stream.GroupID, callerID)
		if err != nil {
			return model.ErrForbidden("you are not a member of this group")
		}
		if err := s.permSvc.RequireCommunication(ctx, model.ActionStartStream, member, callerIsAdmin); err != nil {
			return err
		}
	}

	// Cleanup LiveKit resources if present
	if stream.LiveKitIngressID != nil {
		if err := s.lk.DeleteIngress(ctx, *stream.LiveKitIngressID); err != nil {
			slog.Error("failed to delete ingress", "error", err, "ingress_id", *stream.LiveKitIngressID)
		}
	}
	if stream.LiveKitRoom != nil {
		room, err := s.roomRepo.GetByLiveKitRoom(ctx, *stream.LiveKitRoom)
		if err == nil {
			_ = s.lk.DeleteRoom(ctx, room.LiveKitRoom)
			_ = s.roomRepo.End(ctx, room.ID)
		}
	}

	return s.streamRepo.Delete(ctx, streamID)
}

// StartStream starts ingesting a stream — creates a LiveKit room and
// an ingress or publish token depending on source_type.
func (s *StreamService) StartStream(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) (*model.StreamStartResponse, error) {
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	if stream.IsActive {
		return nil, model.ErrValidation("stream is already active")
	}

	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, stream.GroupID, callerID)
		if err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
		if err := s.permSvc.RequireCommunication(ctx, model.ActionStartStream, member, callerIsAdmin); err != nil {
			return nil, err
		}
	}

	// Create a media room for this stream
	room := &model.MediaRoom{
		Name:            stream.Name,
		RoomType:        model.RoomTypeStream,
		GroupID:         &stream.GroupID,
		CreatedBy:       callerID,
		LiveKitRoom:     livekit.RoomName(model.RoomTypeStream, stream.ID),
		IsActive:        true,
		MaxParticipants: 100,
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	// Create the LiveKit room (0 empty timeout = persistent until explicitly deleted)
	_, err = s.lk.CreateRoom(ctx, room.LiveKitRoom, room.MaxParticipants, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create livekit room: %w", err)
	}

	resp := &model.StreamStartResponse{
		Stream: stream.ToResponse(),
	}

	switch stream.SourceType {
	case model.SourceTypeDeviceCamera, model.SourceTypeScreenShare:
		// Device camera / screen share: generate a publish token for the caller.
		user, err := s.userRepo.GetByID(ctx, callerID)
		if err != nil {
			return nil, err
		}
		token, err := s.lk.GenerateToken(callerID.String(), room.LiveKitRoom, livekit.TokenOptions{
			Name:         user.Username,
			CanPublish:   livekit.BoolPtr(true),
			CanSubscribe: livekit.BoolPtr(false),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to generate publish token: %w", err)
		}
		stream.LiveKitRoom = &room.LiveKitRoom
		resp.Token = token
		resp.URL = s.lkURL

	case model.SourceTypeRTMP:
		ingress, err := s.lk.CreateRTMPIngress(ctx, room.LiveKitRoom, stream.ID.String(), stream.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create RTMP ingress: %w", err)
		}
		ingressID := ingress.IngressId
		streamKey := ingress.StreamKey
		stream.LiveKitIngressID = &ingressID
		stream.StreamKey = &streamKey
		stream.LiveKitRoom = &room.LiveKitRoom
		resp.IngestURL = ingress.Url
		resp.StreamKey = ingress.StreamKey

	case model.SourceTypeWHIP:
		ingress, err := s.lk.CreateWHIPIngress(ctx, room.LiveKitRoom, stream.ID.String(), stream.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create WHIP ingress: %w", err)
		}
		ingressID := ingress.IngressId
		streamKey := ingress.StreamKey
		stream.LiveKitIngressID = &ingressID
		stream.StreamKey = &streamKey
		stream.LiveKitRoom = &room.LiveKitRoom
		resp.IngestURL = ingress.Url
		resp.StreamKey = ingress.StreamKey

	case model.SourceTypeRTSP:
		// RTSP uses an RTMP ingress; the caller must run an FFmpeg bridge.
		ingress, err := s.lk.CreateRTMPIngress(ctx, room.LiveKitRoom, stream.ID.String(), stream.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create RTMP ingress for RTSP bridge: %w", err)
		}
		ingressID := ingress.IngressId
		streamKey := ingress.StreamKey
		stream.LiveKitIngressID = &ingressID
		stream.StreamKey = &streamKey
		stream.LiveKitRoom = &room.LiveKitRoom
		resp.IngestURL = ingress.Url
		resp.StreamKey = ingress.StreamKey
	}

	stream.IsActive = true
	if err := s.streamRepo.Update(ctx, stream); err != nil {
		return nil, err
	}

	resp.Stream = stream.ToResponse()
	s.publishStreamEvent(ctx, stream, "started")

	return resp, nil
}

// StopStream stops an active stream and cleans up LiveKit resources.
func (s *StreamService) StopStream(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) error {
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}

	if !stream.IsActive {
		return model.ErrValidation("stream is not active")
	}

	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, stream.GroupID, callerID)
		if err != nil {
			return model.ErrForbidden("you are not a member of this group")
		}
		if err := s.permSvc.RequireCommunication(ctx, model.ActionStartStream, member, callerIsAdmin); err != nil {
			return err
		}
	}

	if stream.LiveKitIngressID != nil {
		if err := s.lk.DeleteIngress(ctx, *stream.LiveKitIngressID); err != nil {
			slog.Error("failed to delete ingress", "error", err)
		}
	}
	if stream.LiveKitRoom != nil {
		if err := s.lk.DeleteRoom(ctx, *stream.LiveKitRoom); err != nil {
			slog.Error("failed to delete livekit room", "error", err)
		}
		room, err := s.roomRepo.GetByLiveKitRoom(ctx, *stream.LiveKitRoom)
		if err == nil {
			_ = s.roomRepo.End(ctx, room.ID)
		}
	}

	stream.IsActive = false
	stream.LiveKitIngressID = nil
	stream.StreamKey = nil
	if err := s.streamRepo.Update(ctx, stream); err != nil {
		return err
	}

	s.publishStreamEvent(ctx, stream, "stopped")
	return nil
}

// GetViewToken generates a subscribe-only token for watching a stream.
func (s *StreamService) GetViewToken(ctx context.Context, streamID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) (*model.JoinRoomResponse, error) {
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	if !stream.IsActive || stream.LiveKitRoom == nil {
		return nil, model.ErrValidation("stream is not active")
	}

	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, stream.GroupID, callerID)
		if err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
		if err := s.permSvc.RequireCommunication(ctx, model.ActionViewStream, member, callerIsAdmin); err != nil {
			return nil, err
		}
	}

	room, err := s.roomRepo.GetByLiveKitRoom(ctx, *stream.LiveKitRoom)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, callerID)
	if err != nil {
		return nil, err
	}

	token, err := s.lk.GenerateToken(callerID.String(), room.LiveKitRoom, livekit.TokenOptions{
		Name:         user.Username,
		CanPublish:   livekit.BoolPtr(false),
		CanSubscribe: livekit.BoolPtr(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate view token: %w", err)
	}

	return &model.JoinRoomResponse{
		Room:  room.ToResponse(),
		Token: token,
		URL:   s.lkURL,
	}, nil
}

func (s *StreamService) publishStreamEvent(ctx context.Context, stream *model.Stream, eventType string) {
	if s.ps == nil {
		return
	}
	evt := streamEvent{
		StreamID:   stream.ID,
		StreamName: stream.Name,
		GroupID:    stream.GroupID,
		EventType:  eventType,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		slog.Error("stream service: failed to marshal stream event", "error", err)
		return
	}
	channel := fmt.Sprintf("group:%s:streams", stream.GroupID)
	if err := s.ps.Publish(ctx, channel, data); err != nil {
		slog.Error("stream service: failed to publish stream event", "error", err)
	}
}
