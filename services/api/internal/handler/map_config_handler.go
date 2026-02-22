package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/service"
)

// MapConfigHandler handles map configuration endpoints.
type MapConfigHandler struct {
	mapConfigService *service.MapConfigService
}

// NewMapConfigHandler creates a new MapConfigHandler.
func NewMapConfigHandler(mapConfigService *service.MapConfigService) *MapConfigHandler {
	return &MapConfigHandler{mapConfigService: mapConfigService}
}

// GetSettings handles GET /api/v1/map/settings (authenticated)
// Returns the map settings for the client including tile URL, center, zoom, and all configs.
func (h *MapConfigHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.mapConfigService.GetSettings(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, settings)
}

// GetDefaults handles GET /api/v1/map-configs/defaults (admin)
// Returns the server-level environment defaults so admins can see the baseline
// configuration and revert to it if needed.
func (h *MapConfigHandler) GetDefaults(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, h.mapConfigService.GetDefaults())
}

// --------------------------------------------------------------------------
// Admin CRUD
// --------------------------------------------------------------------------

// List handles GET /api/v1/map-configs (admin)
func (h *MapConfigHandler) List(w http.ResponseWriter, r *http.Request) {
	configs, err := h.mapConfigService.List(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.MapConfigResponse, len(configs))
	for i, mc := range configs {
		items[i] = mc.ToResponse()
	}

	JSON(w, http.StatusOK, items)
}

// Create handles POST /api/v1/map-configs (admin)
func (h *MapConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[model.CreateMapConfigRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	mc, err := h.mapConfigService.Create(r.Context(), &req, claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, mc.ToResponse())
}

// Get handles GET /api/v1/map-configs/{id} (admin)
func (h *MapConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid map config id")
		return
	}

	mc, err := h.mapConfigService.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, mc.ToResponse())
}

// Update handles PUT /api/v1/map-configs/{id} (admin)
func (h *MapConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid map config id")
		return
	}

	req, err := Decode[model.UpdateMapConfigRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	mc, err := h.mapConfigService.Update(r.Context(), id, &req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, mc.ToResponse())
}

// Delete handles DELETE /api/v1/map-configs/{id} (admin)
func (h *MapConfigHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid map config id")
		return
	}

	if err := h.mapConfigService.Delete(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
