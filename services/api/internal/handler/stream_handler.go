package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/service"
)

// StreamHandler handles stream and stream key HTTP endpoints.
type StreamHandler struct {
	streamService *service.StreamService
}

// NewStreamHandler creates a new StreamHandler.
func NewStreamHandler(streamService *service.StreamService) *StreamHandler {
	return &StreamHandler{streamService: streamService}
}

// Create handles POST /api/v1/streams
func (h *StreamHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[service.CreateStreamRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	swd, err := h.streamService.Create(r.Context(), claims.UserID, req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, swd.ToResponse())
}

// List handles GET /api/v1/streams?status=live|ended
func (h *StreamHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	status := r.URL.Query().Get("status")
	if status != "" && status != "live" && status != "ended" {
		Error(w, http.StatusBadRequest, "validation_error", "status must be 'live' or 'ended'")
		return
	}

	streams, err := h.streamService.List(r.Context(), claims.UserID, status)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp := make([]interface{}, len(streams))
	for i := range streams {
		resp[i] = streams[i].ToResponse()
	}

	JSON(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/streams/{id}
func (h *StreamHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	streamID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid stream id")
		return
	}

	swd, err := h.streamService.Get(r.Context(), streamID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, swd.ToResponse())
}

// Share handles POST /api/v1/streams/{id}/share
func (h *StreamHandler) Share(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	streamID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid stream id")
		return
	}

	req, err := Decode[service.ShareStreamRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	if err := h.streamService.Share(r.Context(), streamID, claims.UserID, claims.IsAdmin, req); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Unshare handles DELETE /api/v1/streams/{id}/groups/{groupId}
func (h *StreamHandler) Unshare(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	streamID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid stream id")
		return
	}

	groupID, err := uuid.Parse(r.PathValue("groupId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid group id")
		return
	}

	if err := h.streamService.Unshare(r.Context(), streamID, groupID, claims.UserID, claims.IsAdmin); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// End handles POST /api/v1/streams/{id}/end
func (h *StreamHandler) End(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	streamID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid stream id")
		return
	}

	if err := h.streamService.End(r.Context(), streamID, claims.UserID, claims.IsAdmin); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteStream handles DELETE /api/v1/streams/{id}
func (h *StreamHandler) DeleteStream(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	streamID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid stream id")
		return
	}

	if err := h.streamService.Delete(r.Context(), streamID, claims.UserID, claims.IsAdmin); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetLocations handles GET /api/v1/streams/{id}/locations
func (h *StreamHandler) GetLocations(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	streamID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid stream id")
		return
	}

	locations, err := h.streamService.GetLocations(r.Context(), streamID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, locations)
}

// AuthenticateMedia handles POST /api/v1/media/auth (called by MediaMTX)
func (h *StreamHandler) AuthenticateMedia(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[service.MediaAuthRequest](r)
	if err != nil {
		Error(w, http.StatusUnauthorized, "unauthorized", "invalid request")
		return
	}

	if err := h.streamService.AuthenticateMedia(r.Context(), req); err != nil {
		Error(w, http.StatusUnauthorized, "unauthorized", "access denied")
		return
	}

	// 200 OK = allow
	w.WriteHeader(http.StatusOK)
}

// RecordingComplete handles POST /api/v1/media/recording-complete (called by MediaMTX)
func (h *StreamHandler) RecordingComplete(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[service.RecordingCompleteRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	if err := h.streamService.HandleRecordingComplete(r.Context(), req); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ---------------------------------------------------------------------------
// Stream Key endpoints (admin)
// ---------------------------------------------------------------------------

// CreateKey handles POST /api/v1/admin/stream-keys
func (h *StreamHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[service.CreateStreamKeyRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp, err := h.streamService.CreateStreamKey(r.Context(), claims.UserID, req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, resp)
}

// ListKeys handles GET /api/v1/admin/stream-keys
func (h *StreamHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.streamService.ListStreamKeys(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, keys)
}

// UpdateKey handles PUT /api/v1/admin/stream-keys/{id}
func (h *StreamHandler) UpdateKey(w http.ResponseWriter, r *http.Request) {
	keyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid stream key id")
		return
	}

	req, err := Decode[service.UpdateStreamKeyRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp, err := h.streamService.UpdateStreamKey(r.Context(), keyID, req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// DeleteKey handles DELETE /api/v1/admin/stream-keys/{id}
func (h *StreamHandler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	keyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid stream key id")
		return
	}

	if err := h.streamService.DeleteStreamKey(r.Context(), keyID); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
