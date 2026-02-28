package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/pubsub"
	"github.com/sitaware/api/internal/repository"
)

// CreateDrawingRequest contains the fields needed to create a drawing.
type CreateDrawingRequest struct {
	Name    string          `json:"name"`
	GeoJSON json.RawMessage `json:"geojson"`
}

// UpdateDrawingRequest contains the fields that can be updated on a drawing.
type UpdateDrawingRequest struct {
	Name    *string          `json:"name,omitempty"`
	GeoJSON *json.RawMessage `json:"geojson,omitempty"`
}

// ShareDrawingRequest contains the target for sharing a drawing.
type ShareDrawingRequest struct {
	GroupID     *uuid.UUID `json:"group_id,omitempty"`
	RecipientID *uuid.UUID `json:"recipient_id,omitempty"`
}

// DrawingService handles drawing business logic.
type DrawingService struct {
	drawingRepo repository.DrawingRepo
	messageRepo repository.MessageRepo
	groupRepo   repository.GroupRepo
	ps          pubsub.PubSub
	permSvc     *PermissionPolicyService
}

// NewDrawingService creates a new DrawingService.
func NewDrawingService(
	drawingRepo repository.DrawingRepo,
	messageRepo repository.MessageRepo,
	groupRepo repository.GroupRepo,
	ps pubsub.PubSub,
	permSvc *PermissionPolicyService,
) *DrawingService {
	return &DrawingService{
		drawingRepo: drawingRepo,
		messageRepo: messageRepo,
		groupRepo:   groupRepo,
		ps:          ps,
		permSvc:     permSvc,
	}
}

// Create creates a new drawing.
func (s *DrawingService) Create(ctx context.Context, ownerID uuid.UUID, req CreateDrawingRequest) (*model.DrawingWithUser, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, model.ErrValidation("name is required")
	}

	if len(req.GeoJSON) == 0 {
		return nil, model.ErrValidation("geojson is required")
	}

	// Basic GeoJSON validation: must be valid JSON with a "type" field
	var check struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(req.GeoJSON, &check); err != nil || check.Type == "" {
		return nil, model.ErrValidation("geojson must be valid GeoJSON with a type field")
	}

	d := &model.Drawing{
		OwnerID: ownerID,
		Name:    name,
		GeoJSON: req.GeoJSON,
	}
	if err := s.drawingRepo.Create(ctx, d); err != nil {
		return nil, fmt.Errorf("create drawing: %w", err)
	}

	// Re-fetch to get owner username/display_name
	return s.drawingRepo.GetByID(ctx, d.ID)
}

// Get retrieves a drawing by ID. Caller must be the owner, an admin, or
// have been shared the drawing.
func (s *DrawingService) Get(ctx context.Context, drawingID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) (*model.DrawingWithUser, error) {
	dwu, err := s.drawingRepo.GetByID(ctx, drawingID)
	if err != nil {
		return nil, err
	}

	if !callerIsAdmin && dwu.OwnerID != callerID {
		if err := s.checkDrawingAccess(ctx, drawingID, callerID); err != nil {
			return nil, err
		}
	}

	return dwu, nil
}

// ListOwn returns all drawings owned by the caller.
func (s *DrawingService) ListOwn(ctx context.Context, callerID uuid.UUID) ([]model.DrawingWithUser, error) {
	return s.drawingRepo.ListByOwner(ctx, callerID)
}

// ListShared returns drawings shared with the caller by other users.
func (s *DrawingService) ListShared(ctx context.Context, callerID uuid.UUID) ([]model.DrawingWithUser, error) {
	return s.drawingRepo.ListSharedWithUser(ctx, callerID)
}

// Update modifies a drawing. Only the owner (or admin) can update.
// After updating, broadcasts a drawing_updated event to all share targets.
func (s *DrawingService) Update(ctx context.Context, drawingID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool, req UpdateDrawingRequest) (*model.DrawingWithUser, error) {
	dwu, err := s.drawingRepo.GetByID(ctx, drawingID)
	if err != nil {
		return nil, err
	}

	if !callerIsAdmin && dwu.OwnerID != callerID {
		return nil, model.ErrForbidden("you can only edit your own drawings")
	}

	// Apply updates
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, model.ErrValidation("name cannot be empty")
		}
		dwu.Name = name
	}
	if req.GeoJSON != nil {
		if len(*req.GeoJSON) == 0 {
			return nil, model.ErrValidation("geojson cannot be empty")
		}
		var check struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(*req.GeoJSON, &check); err != nil || check.Type == "" {
			return nil, model.ErrValidation("geojson must be valid GeoJSON with a type field")
		}
		dwu.GeoJSON = *req.GeoJSON
	}

	if err := s.drawingRepo.Update(ctx, &dwu.Drawing); err != nil {
		return nil, fmt.Errorf("update drawing: %w", err)
	}

	// Re-fetch for updated timestamps and owner info
	updated, err := s.drawingRepo.GetByID(ctx, drawingID)
	if err != nil {
		return nil, err
	}

	// Broadcast update to share targets
	s.broadcastDrawingUpdate(ctx, updated)

	return updated, nil
}

// Delete removes a drawing. Only the owner (or admin) can delete.
func (s *DrawingService) Delete(ctx context.Context, drawingID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) error {
	dwu, err := s.drawingRepo.GetByID(ctx, drawingID)
	if err != nil {
		return err
	}

	if !callerIsAdmin && dwu.OwnerID != callerID {
		return model.ErrForbidden("you can only delete your own drawings")
	}

	return s.drawingRepo.Delete(ctx, drawingID)
}

// Share creates a message that shares a drawing with a group or user.
// The message has message_type "drawing" and metadata containing the drawing_id.
func (s *DrawingService) Share(ctx context.Context, drawingID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool, req ShareDrawingRequest) (*model.MessageWithUser, error) {
	// Validate target
	if req.GroupID == nil && req.RecipientID == nil {
		return nil, model.ErrValidation("group_id or recipient_id is required")
	}
	if req.GroupID != nil && req.RecipientID != nil {
		return nil, model.ErrValidation("only one of group_id or recipient_id may be set")
	}

	// Verify drawing exists and caller is the owner
	dwu, err := s.drawingRepo.GetByID(ctx, drawingID)
	if err != nil {
		return nil, err
	}
	if dwu.OwnerID != callerID {
		return nil, model.ErrForbidden("you can only share your own drawings")
	}

	// If sharing to a group, verify the caller is a member with share_drawings permission
	if req.GroupID != nil {
		member, err := s.groupRepo.GetMember(ctx, *req.GroupID, callerID)
		if err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
		if err := s.permSvc.RequireCommunication(ctx, model.ActionShareDrawings, member, callerIsAdmin); err != nil {
			return nil, err
		}
	}

	// Build the metadata with drawing_id reference
	meta := struct {
		DrawingID uuid.UUID `json:"drawing_id"`
	}{DrawingID: drawingID}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal drawing metadata: %w", err)
	}
	metaRaw := json.RawMessage(metaBytes)

	// Build content text
	content := fmt.Sprintf("Shared drawing: %s", dwu.Name)

	// Create the message
	msg := &model.Message{
		SenderID:    callerID,
		GroupID:     req.GroupID,
		RecipientID: req.RecipientID,
		Content:     &content,
		MessageType: "drawing",
		Metadata:    &metaRaw,
	}
	if err := s.messageRepo.Create(ctx, msg); err != nil {
		return nil, fmt.Errorf("create share message: %w", err)
	}

	// Re-fetch with user info
	full, err := s.messageRepo.GetByID(ctx, msg.ID)
	if err != nil {
		slog.Error("failed to re-fetch share message", "error", err)
		return nil, err
	}

	// Publish message to Redis for real-time delivery
	s.publishShareMessage(ctx, full)

	return full, nil
}

// ListShares returns the active share targets for a drawing. Only the owner may view.
func (s *DrawingService) ListShares(ctx context.Context, drawingID uuid.UUID, callerID uuid.UUID) ([]model.DrawingShareInfo, error) {
	dwu, err := s.drawingRepo.GetByID(ctx, drawingID)
	if err != nil {
		return nil, err
	}
	if dwu.OwnerID != callerID {
		return nil, model.ErrForbidden("you can only view shares of your own drawings")
	}
	shares, err := s.drawingRepo.ListShares(ctx, drawingID)
	if err != nil {
		return nil, fmt.Errorf("list shares: %w", err)
	}
	if shares == nil {
		shares = []model.DrawingShareInfo{}
	}
	return shares, nil
}

// Unshare revokes a share (by message ID) and sends a notification message.
func (s *DrawingService) Unshare(ctx context.Context, drawingID uuid.UUID, callerID uuid.UUID, messageID uuid.UUID) error {
	dwu, err := s.drawingRepo.GetByID(ctx, drawingID)
	if err != nil {
		return err
	}
	if dwu.OwnerID != callerID {
		return model.ErrForbidden("you can only unshare your own drawings")
	}

	// Fetch the original share message to determine the target
	origMsg, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("get share message: %w", err)
	}

	// Revoke the share
	if err := s.drawingRepo.RevokeShare(ctx, messageID); err != nil {
		return fmt.Errorf("revoke share: %w", err)
	}

	// Send a notification message to the same target
	content := fmt.Sprintf("Drawing unshared: %s", dwu.Name)
	notif := &model.Message{
		SenderID:    callerID,
		GroupID:     origMsg.GroupID,
		RecipientID: origMsg.RecipientID,
		Content:     &content,
		MessageType: "text",
	}
	if err := s.messageRepo.Create(ctx, notif); err != nil {
		slog.Error("failed to create unshare notification", "error", err)
		// Non-fatal: the revocation succeeded, notification is best-effort
	}

	return nil
}

// --------------------------------------------------------------------------
// Internal helpers
// --------------------------------------------------------------------------

// checkDrawingAccess verifies the caller has access to a drawing via sharing.
func (s *DrawingService) checkDrawingAccess(ctx context.Context, drawingID uuid.UUID, callerID uuid.UUID) error {
	groupIDs, userIDs, err := s.drawingRepo.GetShareTargets(ctx, drawingID)
	if err != nil {
		return fmt.Errorf("check drawing access: %w", err)
	}

	// Check direct shares
	for _, uid := range userIDs {
		if uid == callerID {
			return nil
		}
	}

	// Check group shares
	for _, gid := range groupIDs {
		_, err := s.groupRepo.GetMember(ctx, gid, callerID)
		if err == nil {
			return nil
		}
	}

	return model.ErrForbidden("you do not have access to this drawing")
}

// broadcastDrawingUpdate publishes a drawing_updated event to all share
// target channels (groups and direct recipients).
func (s *DrawingService) broadcastDrawingUpdate(ctx context.Context, dwu *model.DrawingWithUser) {
	resp := dwu.ToResponse()
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal drawing for broadcast", "error", err)
		return
	}

	groupIDs, userIDs, err := s.drawingRepo.GetShareTargets(ctx, dwu.ID)
	if err != nil {
		slog.Error("failed to get share targets for broadcast", "error", err, "drawing_id", dwu.ID)
		return
	}

	for _, gid := range groupIDs {
		channel := fmt.Sprintf("group:%s:drawings", gid)
		if err := s.ps.Publish(ctx, channel, data); err != nil {
			slog.Error("failed to publish drawing update to group", "error", err, "channel", channel)
		} else {
			slog.Debug("published drawing update", "channel", channel, "drawing_id", dwu.ID)
		}
	}

	for _, uid := range userIDs {
		channel := fmt.Sprintf("user:%s:drawings", uid)
		if err := s.ps.Publish(ctx, channel, data); err != nil {
			slog.Error("failed to publish drawing update to user", "error", err, "channel", channel)
		} else {
			slog.Debug("published drawing update", "channel", channel, "drawing_id", dwu.ID)
		}
	}
}

// publishShareMessage publishes a share message to Redis for real-time delivery.
func (s *DrawingService) publishShareMessage(ctx context.Context, msg *model.MessageWithUser) {
	resp := msg.ToResponse()
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal share message for broadcast", "error", err)
		return
	}

	if msg.GroupID != nil {
		channel := fmt.Sprintf("group:%s:messages", msg.GroupID)
		if err := s.ps.Publish(ctx, channel, data); err != nil {
			slog.Error("failed to publish share message", "error", err, "channel", channel)
		}
	} else if msg.RecipientID != nil {
		channel := fmt.Sprintf("user:%s:direct", msg.RecipientID)
		if err := s.ps.Publish(ctx, channel, data); err != nil {
			slog.Error("failed to publish share message", "error", err, "channel", channel)
		}
	}
}
