package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/service"
)

// GarminInReachHandler handles HTTP endpoints for Garmin InReach feed management.
type GarminInReachHandler struct {
	svc *service.GarminInReachService
}

// NewGarminInReachHandler creates a new GarminInReachHandler.
func NewGarminInReachHandler(svc *service.GarminInReachService) *GarminInReachHandler {
	return &GarminInReachHandler{svc: svc}
}

// Create handles POST /api/v1/garmin/inreach/feeds
func (h *GarminInReachHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[model.CreateGarminInReachFeedRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	feed, err := h.svc.Create(r.Context(), req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, feed.ToResponse())
}

// List handles GET /api/v1/garmin/inreach/feeds
func (h *GarminInReachHandler) List(w http.ResponseWriter, r *http.Request) {
	feeds, err := h.svc.List(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	resp := make([]model.GarminInReachFeedResponse, len(feeds))
	for i := range feeds {
		resp[i] = feeds[i].ToResponse()
	}
	JSON(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/garmin/inreach/feeds/{id}
func (h *GarminInReachHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid feed id")
		return
	}

	feed, err := h.svc.Get(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, feed.ToResponse())
}

// Update handles PUT /api/v1/garmin/inreach/feeds/{id}
func (h *GarminInReachHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid feed id")
		return
	}

	req, err := Decode[model.UpdateGarminInReachFeedRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	feed, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, feed.ToResponse())
}

// Delete handles DELETE /api/v1/garmin/inreach/feeds/{id}
func (h *GarminInReachHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid feed id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Poll handles POST /api/v1/garmin/inreach/poll
// Triggers an immediate poll of all enabled feeds (admin use / debugging).
func (h *GarminInReachHandler) Poll(w http.ResponseWriter, r *http.Request) {
	results := h.svc.PollAll(r.Context())
	JSON(w, http.StatusOK, results)
}

// Webhook handles POST /api/v1/webhooks/garmin/inreach/{mapshareId}
// Receives Garmin Explore outbound KML data.
func (h *GarminInReachHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	mapShareID := r.PathValue("mapshareId")
	if mapShareID == "" {
		Error(w, http.StatusBadRequest, "validation_error", "mapshare id is required")
		return
	}

	result, err := h.svc.IngestWebhook(r.Context(), mapShareID, r.Body)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, result)
}
