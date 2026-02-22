package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/service"
)

// TerrainConfigHandler handles terrain configuration endpoints.
type TerrainConfigHandler struct {
	terrainConfigService *service.TerrainConfigService
}

// NewTerrainConfigHandler creates a new TerrainConfigHandler.
func NewTerrainConfigHandler(terrainConfigService *service.TerrainConfigService) *TerrainConfigHandler {
	return &TerrainConfigHandler{terrainConfigService: terrainConfigService}
}

// --------------------------------------------------------------------------
// Admin CRUD
// --------------------------------------------------------------------------

// List handles GET /api/v1/terrain-configs (admin)
func (h *TerrainConfigHandler) List(w http.ResponseWriter, r *http.Request) {
	configs, err := h.terrainConfigService.List(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.TerrainConfigResponse, len(configs))
	for i, tc := range configs {
		items[i] = tc.ToResponse()
	}

	JSON(w, http.StatusOK, items)
}

// Create handles POST /api/v1/terrain-configs (admin)
func (h *TerrainConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[model.CreateTerrainConfigRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	tc, err := h.terrainConfigService.Create(r.Context(), &req, claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, tc.ToResponse())
}

// Get handles GET /api/v1/terrain-configs/{id} (admin)
func (h *TerrainConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid terrain config id")
		return
	}

	tc, err := h.terrainConfigService.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, tc.ToResponse())
}

// Update handles PUT /api/v1/terrain-configs/{id} (admin)
func (h *TerrainConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid terrain config id")
		return
	}

	req, err := Decode[model.UpdateTerrainConfigRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	tc, err := h.terrainConfigService.Update(r.Context(), id, &req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, tc.ToResponse())
}

// Delete handles DELETE /api/v1/terrain-configs/{id} (admin)
func (h *TerrainConfigHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid terrain config id")
		return
	}

	if err := h.terrainConfigService.Delete(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
