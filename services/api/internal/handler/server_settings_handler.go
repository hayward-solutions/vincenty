package handler

import (
	"fmt"
	"net/http"

	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// ServerSettingsHandler handles server settings endpoints.
type ServerSettingsHandler struct {
	settingsRepo *repository.ServerSettingsRepository
}

// NewServerSettingsHandler creates a new ServerSettingsHandler.
func NewServerSettingsHandler(settingsRepo *repository.ServerSettingsRepository) *ServerSettingsHandler {
	return &ServerSettingsHandler{settingsRepo: settingsRepo}
}

// GetSettings handles GET /api/v1/server/settings (admin)
func (h *ServerSettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	resp, err := h.buildResponse(r)
	if err != nil {
		HandleError(w, err)
		return
	}
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

	resp, err := h.buildResponse(r)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, resp)
}

func (h *ServerSettingsHandler) buildResponse(r *http.Request) (*model.ServerSettingsResponse, error) {
	mfaSetting, err := h.settingsRepo.Get(r.Context(), "mfa_required")
	if err != nil {
		return nil, fmt.Errorf("get mfa_required setting: %w", err)
	}

	return &model.ServerSettingsResponse{
		MFARequired: mfaSetting.Value == "true",
	}, nil
}
