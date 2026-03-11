package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/middleware"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/service"
)

// MessageHandler handles messaging HTTP endpoints.
type MessageHandler struct {
	messageService *service.MessageService
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

// Send handles POST /api/v1/messages
// Accepts multipart/form-data with fields: content, group_id, recipient_id, lat, lng
// and optional file parts named "files".
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	// Parse multipart form (max 25 MB in memory, rest to temp files)
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid multipart form")
		return
	}

	req := service.SendMessageRequest{
		SenderID:      claims.UserID,
		CallerIsAdmin: claims.IsAdmin,
	}

	// Content (optional if files are present)
	if v := r.FormValue("content"); v != "" {
		req.Content = &v
	}

	// Target: group_id or recipient_id
	if v := r.FormValue("group_id"); v != "" {
		gid, err := uuid.Parse(v)
		if err != nil {
			Error(w, http.StatusBadRequest, "validation_error", "invalid group_id")
			return
		}
		req.GroupID = &gid
	}
	if v := r.FormValue("recipient_id"); v != "" {
		rid, err := uuid.Parse(v)
		if err != nil {
			Error(w, http.StatusBadRequest, "validation_error", "invalid recipient_id")
			return
		}
		req.RecipientID = &rid
	}

	// Optional sender device
	if v := r.FormValue("device_id"); v != "" {
		did, err := uuid.Parse(v)
		if err == nil {
			req.SenderDeviceID = &did
		}
	}

	// Optional location
	if latStr := r.FormValue("lat"); latStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err == nil {
			req.Lat = &lat
		}
	}
	if lngStr := r.FormValue("lng"); lngStr != "" {
		lng, err := strconv.ParseFloat(lngStr, 64)
		if err == nil {
			req.Lng = &lng
		}
	}

	// Files
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		for _, fh := range r.MultipartForm.File["files"] {
			f, err := fh.Open()
			if err != nil {
				Error(w, http.StatusBadRequest, "validation_error", "failed to read uploaded file")
				return
			}
			defer f.Close()

			req.Files = append(req.Files, service.FileUpload{
				Filename:    fh.Filename,
				ContentType: fh.Header.Get("Content-Type"),
				Size:        fh.Size,
				Body:        f,
			})
		}
	}

	msg, err := h.messageService.Send(r.Context(), req)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp := msg.ToResponse()
	JSON(w, http.StatusCreated, resp)
}

// ListGroupMessages handles GET /api/v1/groups/{id}/messages?before=&limit=
func (h *MessageHandler) ListGroupMessages(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	groupID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid group id")
		return
	}

	var before *uuid.UUID
	if v := r.URL.Query().Get("before"); v != "" {
		b, err := uuid.Parse(v)
		if err != nil {
			Error(w, http.StatusBadRequest, "validation_error", "invalid before cursor")
			return
		}
		before = &b
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	messages, err := h.messageService.ListGroupMessages(r.Context(), groupID, claims.UserID, claims.IsAdmin, before, limit)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp := make([]model.MessageResponse, len(messages))
	for i := range messages {
		resp[i] = messages[i].ToResponse()
	}

	JSON(w, http.StatusOK, resp)
}

// ListDirectMessages handles GET /api/v1/messages/direct/{userId}?before=&limit=
func (h *MessageHandler) ListDirectMessages(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	otherUserID, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}

	var before *uuid.UUID
	if v := r.URL.Query().Get("before"); v != "" {
		b, err := uuid.Parse(v)
		if err != nil {
			Error(w, http.StatusBadRequest, "validation_error", "invalid before cursor")
			return
		}
		before = &b
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	messages, err := h.messageService.ListDirectMessages(r.Context(), claims.UserID, otherUserID, before, limit)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp := make([]model.MessageResponse, len(messages))
	for i := range messages {
		resp[i] = messages[i].ToResponse()
	}

	JSON(w, http.StatusOK, resp)
}

// GetMessage handles GET /api/v1/messages/{id}
func (h *MessageHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	messageID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid message id")
		return
	}

	msg, err := h.messageService.GetMessage(r.Context(), messageID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, msg.ToResponse())
}

// DeleteMessage handles DELETE /api/v1/messages/{id}
func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	messageID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid message id")
		return
	}

	if err := h.messageService.DeleteMessage(r.Context(), messageID, claims.UserID, claims.IsAdmin); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DownloadAttachment handles GET /api/v1/attachments/{id}/download
// Streams the file content through the API with appropriate headers.
func (h *MessageHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	attachmentID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid attachment id")
		return
	}

	att, body, err := h.messageService.GetAttachment(r.Context(), attachmentID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}
	defer body.Close()

	// Determine disposition: inline for images so browsers render them,
	// attachment for everything else so they trigger a download.
	disposition := "attachment"
	if strings.HasPrefix(att.ContentType, "image/") {
		disposition = "inline"
	}

	w.Header().Set("Content-Type", att.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, att.Filename))
	if att.SizeBytes > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", att.SizeBytes))
	}

	io.Copy(w, body)
}

// ListDMConversations handles GET /api/v1/messages/conversations
// Returns the distinct users the caller has exchanged DMs with.
func (h *MessageHandler) ListDMConversations(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	conversations, err := h.messageService.ListDMConversations(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, conversations)
}
