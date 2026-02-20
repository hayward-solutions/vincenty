package handler

import (
	"net/http"
	"time"

	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/service"
)

// CotHandler handles CoT HTTP endpoints.
type CotHandler struct {
	cotService *service.CotService
}

// NewCotHandler creates a new CotHandler.
func NewCotHandler(cotService *service.CotService) *CotHandler {
	return &CotHandler{cotService: cotService}
}

// IngestEvents handles POST /api/v1/cot/events
// Accepts CoT XML (single <event> or batch <events>) in the request body.
func (h *CotHandler) IngestEvents(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType != "" &&
		contentType != "application/xml" &&
		contentType != "text/xml" &&
		contentType != "application/octet-stream" {
		Error(w, http.StatusUnsupportedMediaType, "unsupported_media_type",
			"expected Content-Type: application/xml or text/xml")
		return
	}

	result, err := h.cotService.Ingest(r.Context(), r.Body)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, result)
}

// ListEvents handles GET /api/v1/cot/events
// Query params: type, uid, from, to, page, page_size
func (h *CotHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	f := parseCotFilters(r)

	events, total, err := h.cotService.List(r.Context(), f)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp := make([]model.CotEventResponse, len(events))
	for i := range events {
		resp[i] = events[i].ToResponse()
	}

	JSON(w, http.StatusOK, model.ListResponse[model.CotEventResponse]{
		Data:     resp,
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
	})
}

// GetLatestByUID handles GET /api/v1/cot/events/{uid}
func (h *CotHandler) GetLatestByUID(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("uid")
	if uid == "" {
		Error(w, http.StatusBadRequest, "validation_error", "uid is required")
		return
	}

	event, err := h.cotService.GetLatestByUID(r.Context(), uid)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, event.ToResponse())
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func parseCotFilters(r *http.Request) model.CotEventFilters {
	page, pageSize := PaginationParams(r)
	f := model.CotEventFilters{
		EventUID:  r.URL.Query().Get("uid"),
		EventType: r.URL.Query().Get("type"),
		Page:      page,
		PageSize:  pageSize,
	}

	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = &t
		}
	}

	return f
}
