package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// DeviceHandler handles device management endpoints.
type DeviceHandler struct {
	deviceRepo *repository.DeviceRepository
}

// NewDeviceHandler creates a new DeviceHandler.
func NewDeviceHandler(deviceRepo *repository.DeviceRepository) *DeviceHandler {
	return &DeviceHandler{deviceRepo: deviceRepo}
}

// List handles GET /api/v1/users/me/devices (authenticated)
func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	devices, err := h.deviceRepo.ListByUserID(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.DeviceResponse, len(devices))
	for i, d := range devices {
		items[i] = d.ToResponse()
	}

	JSON(w, http.StatusOK, items)
}

// Create handles POST /api/v1/users/me/devices (authenticated)
func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[model.CreateDeviceRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	device := &model.Device{
		UserID:     claims.UserID,
		Name:       req.Name,
		DeviceType: req.DeviceType,
	}

	if err := h.deviceRepo.Create(r.Context(), device); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, device.ToResponse())
}

// Update handles PUT /api/v1/devices/{id} (authenticated, own device only)
func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid device id")
		return
	}

	device, err := h.deviceRepo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	if device.UserID != claims.UserID {
		Error(w, http.StatusForbidden, "forbidden", "you can only update your own devices")
		return
	}

	req, err := Decode[model.UpdateDeviceRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	if req.Name != nil {
		device.Name = *req.Name
	}

	if err := h.deviceRepo.Update(r.Context(), device); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, device.ToResponse())
}

// Delete handles DELETE /api/v1/devices/{id} (authenticated, own device only)
func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid device id")
		return
	}

	device, err := h.deviceRepo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	if device.UserID != claims.UserID {
		Error(w, http.StatusForbidden, "forbidden", "you can only delete your own devices")
		return
	}

	if err := h.deviceRepo.Delete(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
