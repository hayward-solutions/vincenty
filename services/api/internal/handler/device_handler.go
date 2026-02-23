package handler

import (
	"log/slog"
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

// Resolve handles POST /api/v1/users/me/devices/resolve (authenticated).
//
// It attempts to recognise the caller as an existing device via:
//  1. Cookie – an HttpOnly "device_id" cookie set on a previous registration.
//  2. User-Agent heuristic – if the user has exactly one existing device of
//     the same type with the same User-Agent string.
//
// If a match is found the response contains matched=true and the device.
// If no match is found the response contains matched=false and the list of
// the user's existing devices so the client can prompt enrolment.
func (h *DeviceHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	ua := r.UserAgent()
	var uaPtr *string
	if ua != "" {
		uaPtr = &ua
	}

	// Layer 1: cookie
	if cookie, err := r.Cookie("device_id"); err == nil && cookie.Value != "" {
		if id, err := uuid.Parse(cookie.Value); err == nil {
			if device, err := h.deviceRepo.GetByID(r.Context(), id); err == nil && device.UserID == claims.UserID {
				_ = h.deviceRepo.TouchLastSeen(r.Context(), device.ID, uaPtr)
				setDeviceCookie(w, device.ID)
				slog.Info("device resolved via cookie", "device_id", device.ID, "user_id", claims.UserID)
				resp := device.ToResponse()
				JSON(w, http.StatusOK, model.DeviceResolveResponse{Matched: true, Device: &resp})
				return
			}
		}
	}

	// Layer 2: user-agent heuristic (web devices only)
	if ua != "" {
		device, err := h.deviceRepo.FindSingleByUserAgent(r.Context(), claims.UserID, "web", ua)
		if err != nil {
			HandleError(w, err)
			return
		}
		if device != nil {
			_ = h.deviceRepo.TouchLastSeen(r.Context(), device.ID, uaPtr)
			setDeviceCookie(w, device.ID)
			slog.Info("device resolved via user-agent heuristic", "device_id", device.ID, "user_id", claims.UserID)
			resp := device.ToResponse()
			JSON(w, http.StatusOK, model.DeviceResolveResponse{Matched: true, Device: &resp})
			return
		}
	}

	// No match — return existing devices so the client can prompt.
	devices, err := h.deviceRepo.ListByUserID(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}
	items := make([]model.DeviceResponse, len(devices))
	for i, d := range devices {
		items[i] = d.ToResponse()
	}

	slog.Info("device not resolved, returning existing devices", "count", len(items), "user_id", claims.UserID)
	JSON(w, http.StatusOK, model.DeviceResolveResponse{Matched: false, ExistingDevices: items})
}

// Claim handles POST /api/v1/users/me/devices/{id}/claim (authenticated).
//
// The user has chosen to re-use an existing device (e.g. after clearing
// browser data). The endpoint verifies ownership, sets the cookie, and
// updates last_seen_at / user_agent.
func (h *DeviceHandler) Claim(w http.ResponseWriter, r *http.Request) {
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
		Error(w, http.StatusForbidden, "forbidden", "you can only claim your own devices")
		return
	}

	ua := r.UserAgent()
	var uaPtr *string
	if ua != "" {
		uaPtr = &ua
	}

	_ = h.deviceRepo.TouchLastSeen(r.Context(), device.ID, uaPtr)
	setDeviceCookie(w, device.ID)
	slog.Info("device claimed", "device_id", device.ID, "user_id", claims.UserID)
	JSON(w, http.StatusOK, device.ToResponse())
}

// Create handles POST /api/v1/users/me/devices (authenticated).
//
// Creates a brand-new device record. This is called when the user explicitly
// chooses to register a new device (or on first login when they have no
// devices at all). The response sets the device_id cookie.
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

	ua := r.UserAgent()
	var uaPtr *string
	if ua != "" {
		uaPtr = &ua
	}

	device := &model.Device{
		UserID:     claims.UserID,
		Name:       req.Name,
		DeviceType: req.DeviceType,
		UserAgent:  uaPtr,
	}

	if err := h.deviceRepo.Create(r.Context(), device); err != nil {
		HandleError(w, err)
		return
	}

	setDeviceCookie(w, device.ID)
	slog.Info("new device created", "device_id", device.ID, "user_id", claims.UserID)
	JSON(w, http.StatusCreated, device.ToResponse())
}

// setDeviceCookie writes an HttpOnly cookie containing the device ID.
// The cookie is long-lived (10 years) and scoped to the API path.
func setDeviceCookie(w http.ResponseWriter, deviceID uuid.UUID) {
	http.SetCookie(w, &http.Cookie{
		Name:     "device_id",
		Value:    deviceID.String(),
		Path:     "/",
		MaxAge:   10 * 365 * 24 * 60 * 60, // ~10 years
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure is left false so it works in local dev over HTTP.
		// In production behind TLS-terminating proxy this is fine;
		// optionally make it configurable later.
	})
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
	if err := req.Validate(); err != nil {
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

// SetPrimary handles PUT /api/v1/users/me/devices/{id}/primary (authenticated).
//
// Sets the given device as the user's primary device.
func (h *DeviceHandler) SetPrimary(w http.ResponseWriter, r *http.Request) {
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
		Error(w, http.StatusForbidden, "forbidden", "you can only set your own devices as primary")
		return
	}

	if err := h.deviceRepo.SetPrimary(r.Context(), claims.UserID, id); err != nil {
		HandleError(w, err)
		return
	}

	device.IsPrimary = true
	slog.Info("device set as primary", "device_id", id, "user_id", claims.UserID)
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
