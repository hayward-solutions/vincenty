package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/service"
)

// PTTHandler handles push-to-talk channel endpoints.
type PTTHandler struct {
	pttSvc *service.PTTService
}

// NewPTTHandler creates a new PTTHandler.
func NewPTTHandler(pttSvc *service.PTTService) *PTTHandler {
	return &PTTHandler{pttSvc: pttSvc}
}

// CreateChannel handles POST /api/v1/groups/{id}/ptt-channels
func (h *PTTHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
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

	req, err := Decode[model.CreatePTTChannelRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	ch, err := h.pttSvc.CreateChannel(r.Context(), groupID, &req, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, ch.ToResponse())
}

// ListChannels handles GET /api/v1/groups/{id}/ptt-channels
func (h *PTTHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
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

	channels, err := h.pttSvc.ListChannels(r.Context(), groupID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.PTTChannelResponse, len(channels))
	for i, ch := range channels {
		items[i] = ch.ToResponse()
	}

	JSON(w, http.StatusOK, items)
}

// JoinChannel handles POST /api/v1/groups/{id}/ptt-channels/{channelId}/join
func (h *PTTHandler) JoinChannel(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	channelID, err := uuid.Parse(r.PathValue("channelId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid channel id")
		return
	}

	resp, err := h.pttSvc.JoinChannel(r.Context(), channelID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// DeleteChannel handles DELETE /api/v1/groups/{id}/ptt-channels/{channelId}
func (h *PTTHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	channelID, err := uuid.Parse(r.PathValue("channelId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid channel id")
		return
	}

	if err := h.pttSvc.DeleteChannel(r.Context(), channelID, claims.UserID, claims.IsAdmin); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
