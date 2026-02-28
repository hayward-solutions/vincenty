package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"

	"github.com/google/uuid"
	imgexif "github.com/sitaware/api/internal/exif"
	"github.com/sitaware/api/internal/gpx"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/pubsub"
	"github.com/sitaware/api/internal/repository"
	"github.com/sitaware/api/internal/storage"
)

const maxAttachmentSize int64 = 25 * 1024 * 1024 // 25 MB

// FileUpload represents a file attached to a message being sent.
type FileUpload struct {
	Filename    string
	ContentType string
	Size        int64
	Body        io.Reader
}

// SendMessageRequest contains the fields needed to send a message.
type SendMessageRequest struct {
	SenderID       uuid.UUID
	SenderDeviceID *uuid.UUID
	GroupID        *uuid.UUID
	RecipientID    *uuid.UUID
	Content        *string
	Lat            *float64
	Lng            *float64
	Files          []FileUpload
	CallerIsAdmin  bool
}

// MessageService handles messaging business logic.
type MessageService struct {
	messageRepo repository.MessageRepo
	groupRepo   repository.GroupRepo
	storage     storage.Storage
	ps          pubsub.PubSub
	permSvc     *PermissionPolicyService
}

// NewMessageService creates a new MessageService.
func NewMessageService(
	messageRepo repository.MessageRepo,
	groupRepo repository.GroupRepo,
	storage storage.Storage,
	ps pubsub.PubSub,
	permSvc *PermissionPolicyService,
) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		groupRepo:   groupRepo,
		storage:     storage,
		ps:          ps,
		permSvc:     permSvc,
	}
}

// Send creates a message, uploads any attachments, parses GPX files, and publishes to Redis.
func (s *MessageService) Send(ctx context.Context, req SendMessageRequest) (*model.MessageWithUser, error) {
	// Validate target
	if req.GroupID == nil && req.RecipientID == nil {
		return nil, model.ErrValidation("group_id or recipient_id is required")
	}
	if req.GroupID != nil && req.RecipientID != nil {
		return nil, model.ErrValidation("only one of group_id or recipient_id may be set")
	}
	// Must have content or at least one file
	hasContent := req.Content != nil && strings.TrimSpace(*req.Content) != ""
	if !hasContent && len(req.Files) == 0 {
		return nil, model.ErrValidation("content or at least one file is required")
	}

	// Permission check: sender must be a group member with the appropriate permission
	if req.GroupID != nil {
		member, err := s.groupRepo.GetMember(ctx, *req.GroupID, req.SenderID)
		if err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
		action := model.ActionSendMessages
		if len(req.Files) > 0 {
			action = model.ActionSendAttachments
		}
		if err := s.permSvc.RequireCommunication(ctx, action, member, req.CallerIsAdmin); err != nil {
			return nil, err
		}
	}

	// Validate file sizes
	for _, f := range req.Files {
		if f.Size > maxAttachmentSize {
			return nil, model.ErrValidation(fmt.Sprintf("file %q exceeds 25 MB limit", f.Filename))
		}
	}

	// Determine message type
	messageType := "text"
	if len(req.Files) > 0 {
		messageType = "file"
	}

	// Check if any file is GPX — if so, parse it and store GeoJSON in metadata.
	// We buffer GPX files so we can parse and then still upload.
	var metadata *json.RawMessage
	for i := range req.Files {
		if isGPXFile(req.Files[i].Filename, req.Files[i].ContentType) {
			messageType = "gpx"
			buf, err := io.ReadAll(req.Files[i].Body)
			if err != nil {
				return nil, fmt.Errorf("read gpx file: %w", err)
			}
			geojson, err := gpx.Parse(bytes.NewReader(buf))
			if err != nil {
				slog.Warn("failed to parse GPX, storing as regular file", "error", err, "filename", req.Files[i].Filename)
				messageType = "file"
			} else {
				metadata = &geojson
			}
			// Replace Body with a fresh reader so Upload still works
			req.Files[i].Body = bytes.NewReader(buf)
			break // only parse the first GPX
		}
	}

	// Extract EXIF GPS from image attachments (only if we don't already have GPX metadata).
	if metadata == nil {
		type exifEntry struct {
			AttachmentIdx int      `json:"attachment_id"` // index, replaced with real ID after creation
			Lat           float64  `json:"lat"`
			Lng           float64  `json:"lng"`
			Altitude      *float64 `json:"altitude,omitempty"`
			TakenAt       *string  `json:"taken_at,omitempty"`
		}

		var exifLocs []exifEntry
		for i := range req.Files {
			ct := strings.ToLower(req.Files[i].ContentType)
			if !strings.HasPrefix(ct, "image/") {
				continue
			}

			// Buffer the image so we can read EXIF and still upload afterward
			buf, err := io.ReadAll(req.Files[i].Body)
			if err != nil {
				slog.Warn("failed to read image for EXIF", "error", err, "filename", req.Files[i].Filename)
				continue
			}

			loc := imgexif.ExtractGPS(bytes.NewReader(buf))
			if loc != nil {
				entry := exifEntry{
					AttachmentIdx: i,
					Lat:           loc.Lat,
					Lng:           loc.Lng,
					Altitude:      loc.Altitude,
				}
				if loc.TakenAt != nil {
					ts := loc.TakenAt.UTC().Format("2006-01-02T15:04:05Z")
					entry.TakenAt = &ts
				}
				exifLocs = append(exifLocs, entry)
			}

			// Replace Body with fresh reader so upload still works
			req.Files[i].Body = bytes.NewReader(buf)
		}

		if len(exifLocs) > 0 {
			wrapper := struct {
				ExifLocations []exifEntry `json:"exif_locations"`
			}{ExifLocations: exifLocs}
			raw, err := json.Marshal(wrapper)
			if err == nil {
				jrm := json.RawMessage(raw)
				metadata = &jrm
			}
		}
	}

	// Create message record
	msg := &model.Message{
		SenderID:       req.SenderID,
		SenderDeviceID: req.SenderDeviceID,
		GroupID:        req.GroupID,
		RecipientID:    req.RecipientID,
		Content:        req.Content,
		MessageType:    messageType,
		Lat:            req.Lat,
		Lng:            req.Lng,
		Metadata:       metadata,
	}
	if err := s.messageRepo.Create(ctx, msg); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	// Upload attachments to S3 and create attachment records
	var attachments []model.Attachment
	for i := range req.Files {
		f := &req.Files[i]
		objectKey := fmt.Sprintf("attachments/%s/%d_%s", msg.ID, i, f.Filename)

		if err := s.storage.Upload(ctx, objectKey, f.Body, f.ContentType, f.Size); err != nil {
			slog.Error("failed to upload attachment", "error", err, "key", objectKey)
			continue
		}

		att := &model.Attachment{
			MessageID:   msg.ID,
			Filename:    f.Filename,
			ContentType: f.ContentType,
			SizeBytes:   f.Size,
			ObjectKey:   objectKey,
		}
		if err := s.messageRepo.CreateAttachment(ctx, att); err != nil {
			slog.Error("failed to save attachment record", "error", err, "key", objectKey)
			continue
		}
		attachments = append(attachments, *att)
	}

	// Build the full MessageWithUser for the response and broadcast
	mwu := &model.MessageWithUser{
		Message:     *msg,
		Attachments: attachments,
	}
	// We don't have the sender's username/display_name in hand — re-fetch from DB.
	full, err := s.messageRepo.GetByID(ctx, msg.ID)
	if err != nil {
		slog.Error("failed to re-fetch message after create", "error", err)
		// Return what we have — the caller still gets a valid response
		return mwu, nil
	}

	// Publish to Redis for real-time delivery
	s.publishMessage(ctx, full)

	return full, nil
}

// ListGroupMessages returns messages for a group with cursor-based pagination.
// Caller must be a group member with read_messages permission.
func (s *MessageService) ListGroupMessages(ctx context.Context, groupID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool, before *uuid.UUID, limit int) ([]model.MessageWithUser, error) {
	member, err := s.groupRepo.GetMember(ctx, groupID, callerID)
	if err != nil {
		return nil, model.ErrForbidden("you are not a member of this group")
	}
	if err := s.permSvc.RequireCommunication(ctx, model.ActionReadMessages, member, callerIsAdmin); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.messageRepo.ListByGroup(ctx, groupID, before, limit)
}

// ListDirectMessages returns DMs between the caller and another user with cursor-based pagination.
func (s *MessageService) ListDirectMessages(ctx context.Context, callerID, otherUserID uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.messageRepo.ListDirect(ctx, callerID, otherUserID, before, limit)
}

// DMConversation represents a user the caller has exchanged DMs with.
type DMConversation struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
}

// ListDMConversations returns the distinct users the caller has DM history with.
func (s *MessageService) ListDMConversations(ctx context.Context, callerID uuid.UUID) ([]DMConversation, error) {
	partners, err := s.messageRepo.ListDMPartners(ctx, callerID)
	if err != nil {
		return nil, err
	}

	result := make([]DMConversation, len(partners))
	for i, p := range partners {
		dn := ""
		if p.DisplayName != nil {
			dn = *p.DisplayName
		}
		result[i] = DMConversation{
			UserID:      p.UserID,
			Username:    p.Username,
			DisplayName: dn,
		}
	}
	return result, nil
}

// GetMessage retrieves a single message by ID.
// Caller must be sender, recipient, or a group member with read_messages permission.
func (s *MessageService) GetMessage(ctx context.Context, messageID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) (*model.MessageWithUser, error) {
	msg, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	if err := s.checkMessageAccess(ctx, msg, callerID, callerIsAdmin); err != nil {
		return nil, err
	}

	return msg, nil
}

// DeleteMessage deletes a message. Only the sender or a server admin who is
// a group member can delete messages. Also cleans up S3 objects.
func (s *MessageService) DeleteMessage(ctx context.Context, messageID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) error {
	msg, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	// Sender can always delete their own messages
	if msg.SenderID != callerID {
		// Non-sender needs admin status AND group membership to delete
		if !callerIsAdmin {
			return model.ErrForbidden("you can only delete your own messages")
		}
		// Admin must still be a group member
		if msg.GroupID != nil {
			if _, err := s.groupRepo.GetMember(ctx, *msg.GroupID, callerID); err != nil {
				return model.ErrForbidden("you must be a group member to delete messages")
			}
		}
	}

	// Get attachment keys before deleting the message (cascade will remove attachment rows)
	keys, err := s.messageRepo.GetAttachmentObjectKeys(ctx, messageID)
	if err != nil {
		slog.Error("failed to get attachment keys for cleanup", "error", err, "message_id", messageID)
	}

	if err := s.messageRepo.Delete(ctx, messageID); err != nil {
		return err
	}

	// Clean up S3 objects (best-effort)
	for _, key := range keys {
		if err := s.storage.Delete(ctx, key); err != nil {
			slog.Error("failed to delete S3 object", "error", err, "key", key)
		}
	}

	return nil
}

// GetAttachment retrieves an attachment and downloads the file content from S3.
// Caller must have access to the parent message. The returned io.ReadCloser
// must be closed by the caller.
func (s *MessageService) GetAttachment(ctx context.Context, attachmentID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) (*model.Attachment, io.ReadCloser, error) {
	att, err := s.messageRepo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, err
	}

	// Check access to the parent message
	msg, err := s.messageRepo.GetByID(ctx, att.MessageID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.checkMessageAccess(ctx, msg, callerID, callerIsAdmin); err != nil {
		return nil, nil, err
	}

	// Download file content from S3
	body, _, _, err := s.storage.Download(ctx, att.ObjectKey)
	if err != nil {
		return nil, nil, fmt.Errorf("download attachment: %w", err)
	}

	return att, body, nil
}

// --------------------------------------------------------------------------
// Internal helpers
// --------------------------------------------------------------------------

// checkMessageAccess verifies the caller can access a message.
func (s *MessageService) checkMessageAccess(ctx context.Context, msg *model.MessageWithUser, callerID uuid.UUID, callerIsAdmin bool) error {
	// Sender always has access
	if msg.SenderID == callerID {
		return nil
	}
	// DM recipient has access
	if msg.RecipientID != nil && *msg.RecipientID == callerID {
		return nil
	}
	// Group member with read_messages permission
	if msg.GroupID != nil {
		member, err := s.groupRepo.GetMember(ctx, *msg.GroupID, callerID)
		if err != nil {
			return model.ErrForbidden("you do not have access to this message")
		}
		return s.permSvc.RequireCommunication(ctx, model.ActionReadMessages, member, callerIsAdmin)
	}
	return model.ErrForbidden("you do not have access to this message")
}

// publishMessage publishes a message to Redis for real-time WebSocket delivery.
func (s *MessageService) publishMessage(ctx context.Context, msg *model.MessageWithUser) {
	resp := msg.ToResponse()
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal message for broadcast", "error", err)
		return
	}

	if msg.GroupID != nil {
		channel := fmt.Sprintf("group:%s:messages", msg.GroupID)
		if err := s.ps.Publish(ctx, channel, data); err != nil {
			slog.Error("failed to publish group message", "error", err, "channel", channel)
		} else {
			slog.Debug("published group message", "channel", channel, "message_id", msg.ID)
		}
	} else if msg.RecipientID != nil {
		channel := fmt.Sprintf("user:%s:direct", msg.RecipientID)
		if err := s.ps.Publish(ctx, channel, data); err != nil {
			slog.Error("failed to publish direct message", "error", err, "channel", channel)
		} else {
			slog.Debug("published direct message", "channel", channel, "message_id", msg.ID)
		}
	}
}

// isGPXFile checks if a file is a GPX file by extension or content type.
func isGPXFile(filename, contentType string) bool {
	ext := strings.ToLower(path.Ext(filename))
	return ext == ".gpx" || contentType == "application/gpx+xml"
}
