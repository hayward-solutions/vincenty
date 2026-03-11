package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/service"
)

// RecordingHandler handles recording endpoints.
type RecordingHandler struct {
	recSvc *service.RecordingService
}

// NewRecordingHandler creates a new RecordingHandler.
func NewRecordingHandler(recSvc *service.RecordingService) *RecordingHandler {
	return &RecordingHandler{recSvc: recSvc}
}

// StartRecording handles POST /api/v1/streams/{id}/recordings/start
func (h *RecordingHandler) StartRecording(w http.ResponseWriter, r *http.Request) {
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

	rec, err := h.recSvc.StartStreamRecording(r.Context(), streamID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, rec.ToResponse(""))
}

// StopRecording handles POST /api/v1/recordings/{id}/stop
func (h *RecordingHandler) StopRecording(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	recID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid recording id")
		return
	}

	rec, err := h.recSvc.StopRecording(r.Context(), recID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, rec.ToResponse(""))
}

// GetRecording handles GET /api/v1/recordings/{id}
func (h *RecordingHandler) GetRecording(w http.ResponseWriter, r *http.Request) {
	recID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid recording id")
		return
	}

	rec, err := h.recSvc.GetByID(r.Context(), recID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, rec.ToResponse(""))
}

// ListByStream handles GET /api/v1/streams/{id}/recordings
func (h *RecordingHandler) ListByStream(w http.ResponseWriter, r *http.Request) {
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

	recordings, err := h.recSvc.ListByStream(r.Context(), streamID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.RecordingResponse, len(recordings))
	for i, rec := range recordings {
		items[i] = rec.ToResponse("")
	}

	JSON(w, http.StatusOK, items)
}
