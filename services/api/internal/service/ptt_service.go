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

// pttFloorEvent is published to group:uuid:ptt channels.
type pttFloorEvent struct {
	ChannelID  uuid.UUID `json:"channel_id"`
	EventType  string    `json:"event_type"` // "floor_granted", "floor_released"
	HolderID   uuid.UUID `json:"holder_id,omitempty"`
	HolderName string    `json:"holder_name,omitempty"`
}

// PTTService handles push-to-talk channel business logic.
type PTTService struct {
	pttRepo   repository.PTTChannelRepo
	roomRepo  repository.MediaRoomRepo
	groupRepo repository.GroupRepo
	userRepo  repository.UserRepo
	permSvc   *PermissionPolicyService
	lk        *livekit.Client
	lkURL     string
	ps        pubsub.PubSub
}

// NewPTTService creates a new PTTService.
func NewPTTService(
	pttRepo repository.PTTChannelRepo,
	roomRepo repository.MediaRoomRepo,
	groupRepo repository.GroupRepo,
	userRepo repository.UserRepo,
	permSvc *PermissionPolicyService,
	lk *livekit.Client,
	lkURL string,
	ps pubsub.PubSub,
) *PTTService {
	return &PTTService{
		pttRepo:   pttRepo,
		roomRepo:  roomRepo,
		groupRepo: groupRepo,
		userRepo:  userRepo,
		permSvc:   permSvc,
		lk:        lk,
		lkURL:     lkURL,
		ps:        ps,
	}
}

// CreateChannel creates a new PTT channel for a group.
func (s *PTTService) CreateChannel(ctx context.Context, groupID uuid.UUID, req *model.CreatePTTChannelRequest, callerID uuid.UUID, callerIsAdmin bool) (*model.PTTChannel, error) {
	// Verify group exists
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// Check permission — only group admins or server admins
	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, groupID, callerID)
		if err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
		if !member.IsGroupAdmin {
			return nil, model.ErrForbidden("only group admins can create PTT channels")
		}
	}

	// Create the backing LiveKit room (persistent, no empty timeout)
	roomID := uuid.New()
	lkRoomName := livekit.RoomName(model.RoomTypePTT, roomID)

	room := &model.MediaRoom{
		ID:              roomID,
		Name:            fmt.Sprintf("PTT - %s - %s", group.Name, req.Name),
		RoomType:        model.RoomTypePTT,
		GroupID:         &groupID,
		CreatedBy:       callerID,
		LiveKitRoom:     lkRoomName,
		IsActive:        true,
		MaxParticipants: 100,
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	// Create LiveKit room (0 empty timeout = persistent)
	_, err = s.lk.CreateRoom(ctx, lkRoomName, 100, 0)
	if err != nil {
		slog.Error("failed to create livekit room for PTT channel", "error", err)
		return nil, fmt.Errorf("failed to create PTT room: %w", err)
	}

	// Create the PTT channel record
	ch := &model.PTTChannel{
		GroupID:   groupID,
		RoomID:    room.ID,
		Name:      req.Name,
		IsDefault: req.IsDefault,
	}

	if err := s.pttRepo.Create(ctx, ch); err != nil {
		return nil, err
	}

	return ch, nil
}

// ListChannels lists PTT channels for a group.
func (s *PTTService) ListChannels(ctx context.Context, groupID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) ([]model.PTTChannel, error) {
	if !callerIsAdmin {
		if _, err := s.groupRepo.GetMember(ctx, groupID, callerID); err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
	}
	return s.pttRepo.ListByGroupID(ctx, groupID)
}

// JoinChannel generates a token for a user to join a PTT channel.
// Users join with mic muted by default — PTT is client-controlled.
func (s *PTTService) JoinChannel(ctx context.Context, channelID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) (*model.JoinPTTChannelResponse, error) {
	ch, err := s.pttRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	// Check group membership and PTT permission
	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, ch.GroupID, callerID)
		if err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
		if err := s.permSvc.RequireCommunication(ctx, model.ActionUsePTT, member, callerIsAdmin); err != nil {
			return nil, err
		}
	}

	// Get the backing media room
	room, err := s.roomRepo.GetByID(ctx, ch.RoomID)
	if err != nil {
		return nil, err
	}

	if !room.IsActive {
		return nil, model.ErrValidation("PTT channel is not active")
	}

	user, err := s.userRepo.GetByID(ctx, callerID)
	if err != nil {
		return nil, err
	}

	// Generate token with audio publish permission (video disabled)
	token, err := s.lk.GenerateToken(callerID.String(), room.LiveKitRoom, livekit.TokenOptions{
		Name:           user.Username,
		CanPublish:     livekit.BoolPtr(true), // needed for PTT mic
		CanSubscribe:   livekit.BoolPtr(true), // hear others
		CanPublishData: livekit.BoolPtr(true), // for floor control data messages
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate PTT token: %w", err)
	}

	// Record participation
	_ = s.roomRepo.AddParticipant(ctx, &model.MediaRoomParticipant{
		RoomID: room.ID,
		UserID: callerID,
		Role:   model.ParticipantRoleParticipant,
	})

	return &model.JoinPTTChannelResponse{
		Channel: ch.ToResponse(),
		Token:   token,
		URL:     s.lkURL,
	}, nil
}

// DeleteChannel removes a PTT channel and its backing room.
func (s *PTTService) DeleteChannel(ctx context.Context, channelID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) error {
	ch, err := s.pttRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}

	// Check permission
	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, ch.GroupID, callerID)
		if err != nil {
			return model.ErrForbidden("you are not a member of this group")
		}
		if !member.IsGroupAdmin {
			return model.ErrForbidden("only group admins can delete PTT channels")
		}
	}

	// Delete the LiveKit room
	room, err := s.roomRepo.GetByID(ctx, ch.RoomID)
	if err == nil {
		if err := s.lk.DeleteRoom(ctx, room.LiveKitRoom); err != nil {
			slog.Error("failed to delete livekit room for PTT channel", "error", err)
		}
		_ = s.roomRepo.End(ctx, room.ID)
	}

	return s.pttRepo.Delete(ctx, channelID)
}

// RequestFloor handles a PTT floor request from a user.
func (s *PTTService) RequestFloor(ctx context.Context, channelID uuid.UUID, callerID uuid.UUID) error {
	ch, err := s.pttRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}

	user, err := s.userRepo.GetByID(ctx, callerID)
	if err != nil {
		return err
	}

	holderName := user.Username
	if user.DisplayName != nil {
		holderName = *user.DisplayName
	}

	// For v1, simple floor grant — first come, first served
	// TODO: add Redis-based floor arbitration with TTL
	s.publishFloorEvent(ctx, ch.GroupID, pttFloorEvent{
		ChannelID:  channelID,
		EventType:  "floor_granted",
		HolderID:   callerID,
		HolderName: holderName,
	})

	return nil
}

// ReleaseFloor handles a PTT floor release.
func (s *PTTService) ReleaseFloor(ctx context.Context, channelID uuid.UUID, callerID uuid.UUID) error {
	ch, err := s.pttRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}

	s.publishFloorEvent(ctx, ch.GroupID, pttFloorEvent{
		ChannelID: channelID,
		EventType: "floor_released",
		HolderID:  callerID,
	})

	return nil
}

func (s *PTTService) publishFloorEvent(ctx context.Context, groupID uuid.UUID, evt pttFloorEvent) {
	if s.ps == nil {
		return
	}
	data, err := json.Marshal(evt)
	if err != nil {
		slog.Error("ptt service: failed to marshal floor event", "error", err)
		return
	}
	channel := fmt.Sprintf("group:%s:ptt", groupID)
	if err := s.ps.Publish(ctx, channel, data); err != nil {
		slog.Error("ptt service: failed to publish floor event", "error", err)
	}
}
