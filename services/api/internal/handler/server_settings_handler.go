package handler

import (
	"context"
	"net/http"

	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// ServerSettingsHandler handles server settings endpoints.
type ServerSettingsHandler struct {
	settingsRepo repository.ServerSettingsRepo
}

// NewServerSettingsHandler creates a new ServerSettingsHandler.
func NewServerSettingsHandler(settingsRepo repository.ServerSettingsRepo) *ServerSettingsHandler {
	return &ServerSettingsHandler{settingsRepo: settingsRepo}
}

// GetSettings handles GET /api/v1/server/settings (admin)
func (h *ServerSettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	resp := h.buildResponse(r.Context())
	JSON(w, http.StatusOK, resp)
}

// UpdateSettings handles PUT /api/v1/server/settings (admin)
func (h *ServerSettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[model.UpdateServerSettingsRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	if req.MFARequired != nil {
		val := "false"
		if *req.MFARequired {
			val = "true"
		}
		if err := h.settingsRepo.Set(r.Context(), "mfa_required", val); err != nil {
			HandleError(w, err)
			return
		}
	}

	if req.MapboxAccessToken != nil {
		if err := h.settingsRepo.Set(r.Context(), "mapbox_access_token", *req.MapboxAccessToken); err != nil {
			HandleError(w, err)
			return
		}
	}

	if req.GoogleMapsApiKey != nil {
		if err := h.settingsRepo.Set(r.Context(), "google_maps_api_key", *req.GoogleMapsApiKey); err != nil {
			HandleError(w, err)
			return
		}
	}

	resp := h.buildResponse(r.Context())
	JSON(w, http.StatusOK, resp)
}

// buildResponse assembles the full server settings response from individual
// key-value entries. Missing keys default to their zero value.
func (h *ServerSettingsHandler) buildResponse(ctx context.Context) *model.ServerSettingsResponse {
	return &model.ServerSettingsResponse{
		MFARequired:       h.getSettingValue(ctx, "mfa_required") == "true",
		MapboxAccessToken: h.getSettingValue(ctx, "mapbox_access_token"),
		GoogleMapsApiKey:  h.getSettingValue(ctx, "google_maps_api_key"),
	}
}

// getSettingValue returns the value for a server setting key, or an empty
// string if the key does not exist.
func (h *ServerSettingsHandler) getSettingValue(ctx context.Context, key string) string {
	s, err := h.settingsRepo.Get(ctx, key)
	if err != nil {
		return ""
	}
	return s.Value
}
