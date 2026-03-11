package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/service"
)

// StreamHandler handles stream endpoints.
type StreamHandler struct {
	streamSvc *service.StreamService
}

// NewStreamHandler creates a new StreamHandler.
func NewStreamHandler(streamSvc *service.StreamService) *StreamHandler {
	return &StreamHandler{streamSvc: streamSvc}
}

// ListStreams handles GET /api/v1/streams
func (h *StreamHandler) ListStreams(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	streams, err := h.streamSvc.ListActive(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.StreamResponse, len(streams))
	for i, s := range streams {
		items[i] = s.ToResponse()
	}

	JSON(w, http.StatusOK, items)
}

// CreateStream handles POST /api/v1/streams
func (h *StreamHandler) CreateStream(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[model.CreateStreamRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	stream, err := h.streamSvc.Create(r.Context(), &req, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, stream.ToResponse())
}

// GetStream handles GET /api/v1/streams/{id}
func (h *StreamHandler) GetStream(w http.ResponseWriter, r *http.Request) {
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

	stream, err := h.streamSvc.GetByID(r.Context(), streamID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, stream.ToResponse())
}

// UpdateStream handles PUT /api/v1/streams/{id}
func (h *StreamHandler) UpdateStream(w http.ResponseWriter, r *http.Request) {
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

	req, err := Decode[model.UpdateStreamRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	stream, err := h.streamSvc.Update(r.Context(), streamID, &req, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, stream.ToResponse())
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

	if err := h.streamSvc.Delete(r.Context(), streamID, claims.UserID, claims.IsAdmin); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StartStream handles POST /api/v1/streams/{id}/start
func (h *StreamHandler) StartStream(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.streamSvc.StartStream(r.Context(), streamID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// StopStream handles POST /api/v1/streams/{id}/stop
func (h *StreamHandler) StopStream(w http.ResponseWriter, r *http.Request) {
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

	if err := h.streamSvc.StopStream(r.Context(), streamID, claims.UserID, claims.IsAdmin); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ViewStream handles GET /api/v1/streams/{id}/view
func (h *StreamHandler) ViewStream(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.streamSvc.GetViewToken(r.Context(), streamID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// ListGroupStreams handles GET /api/v1/groups/{id}/streams
func (h *StreamHandler) ListGroupStreams(w http.ResponseWriter, r *http.Request) {
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

	streams, err := h.streamSvc.ListByGroup(r.Context(), groupID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.StreamResponse, len(streams))
	for i, s := range streams {
		items[i] = s.ToResponse()
	}

	JSON(w, http.StatusOK, items)
}
