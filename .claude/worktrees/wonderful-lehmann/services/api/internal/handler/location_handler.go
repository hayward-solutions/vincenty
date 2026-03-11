package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/gpx"
	"github.com/vincenty/api/internal/middleware"
	"github.com/vincenty/api/internal/service"
)

// LocationHandler handles location query endpoints.
type LocationHandler struct {
	locationService *service.LocationService
}

// NewLocationHandler creates a new LocationHandler.
func NewLocationHandler(locationService *service.LocationService) *LocationHandler {
	return &LocationHandler{locationService: locationService}
}

// parseOptionalDeviceID parses the optional device_id query parameter.
func parseOptionalDeviceID(r *http.Request) (*uuid.UUID, error) {
	v := r.URL.Query().Get("device_id")
	if v == "" {
		return nil, nil
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return nil, fmt.Errorf("invalid device_id")
	}
	return &id, nil
}

// GetGroupHistory handles GET /api/v1/groups/{id}/locations/history?from=&to=
// Returns location history for a group within a time range.
func (h *LocationHandler) GetGroupHistory(w http.ResponseWriter, r *http.Request) {
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

	// Parse time range (defaults: last 1 hour)
	now := time.Now()
	from := now.Add(-1 * time.Hour)
	to := now

	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		} else {
			Error(w, http.StatusBadRequest, "validation_error", "from must be RFC3339 format")
			return
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		} else {
			Error(w, http.StatusBadRequest, "validation_error", "to must be RFC3339 format")
			return
		}
	}

	// Limit time range to 24 hours max
	if to.Sub(from) > 24*time.Hour {
		Error(w, http.StatusBadRequest, "validation_error", "time range must not exceed 24 hours")
		return
	}

	records, err := h.locationService.GetGroupHistory(r.Context(), groupID, claims.UserID, claims.IsAdmin, from, to)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, records)
}

// GetMyHistory handles GET /api/v1/users/me/locations/history?from=&to=&device_id=
// Returns the caller's own location history within a time range.
// Optional device_id param filters to a single device.
func (h *LocationHandler) GetMyHistory(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	now := time.Now()
	from := now.Add(-1 * time.Hour)
	to := now

	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		} else {
			Error(w, http.StatusBadRequest, "validation_error", "from must be RFC3339 format")
			return
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		} else {
			Error(w, http.StatusBadRequest, "validation_error", "to must be RFC3339 format")
			return
		}
	}

	if to.Sub(from) > 24*time.Hour {
		Error(w, http.StatusBadRequest, "validation_error", "time range must not exceed 24 hours")
		return
	}

	deviceID, err := parseOptionalDeviceID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	records, err := h.locationService.GetMyHistory(r.Context(), claims.UserID, from, to, deviceID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, records)
}

// ExportGPX handles GET /api/v1/users/me/locations/export?from=&to=&device_id=
// Returns the caller's own location history as a GPX file download.
// Optional device_id param filters to a single device.
func (h *LocationHandler) ExportGPX(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	now := time.Now()
	from := now.Add(-1 * time.Hour)
	to := now

	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		} else {
			Error(w, http.StatusBadRequest, "validation_error", "from must be RFC3339 format")
			return
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		} else {
			Error(w, http.StatusBadRequest, "validation_error", "to must be RFC3339 format")
			return
		}
	}

	if to.Sub(from) > 24*time.Hour {
		Error(w, http.StatusBadRequest, "validation_error", "time range must not exceed 24 hours")
		return
	}

	deviceID, err := parseOptionalDeviceID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	records, err := h.locationService.GetMyHistory(r.Context(), claims.UserID, from, to, deviceID)
	if err != nil {
		HandleError(w, err)
		return
	}

	if len(records) == 0 {
		Error(w, http.StatusNotFound, "not_found", "no location data for the specified time range")
		return
	}

	// Convert to GPX track points
	points := make([]gpx.TrackPoint, len(records))
	for i, rec := range records {
		points[i] = gpx.TrackPoint{
			Lat:       rec.Lat,
			Lng:       rec.Lng,
			Elevation: rec.Altitude,
			Time:      rec.RecordedAt,
		}
	}

	name := fmt.Sprintf("Vincenty Track %s", from.UTC().Format("2006-01-02"))
	data, err := gpx.Generate(name, points)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal_error", "failed to generate GPX")
		return
	}

	ts := from.UTC().Format("20060102") + "_" + to.UTC().Format("20060102")
	w.Header().Set("Content-Type", "application/gpx+xml")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="track_%s.gpx"`, ts))
	w.Write(data)
}

// GetVisibleHistory handles GET /api/v1/locations/history?from=&to=
// Returns location history for all users visible to the caller.
// Admins see all users; non-admins see users who share a group with them.
func (h *LocationHandler) GetVisibleHistory(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	now := time.Now()
	from := now.Add(-1 * time.Hour)
	to := now

	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		} else {
			Error(w, http.StatusBadRequest, "validation_error", "from must be RFC3339 format")
			return
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		} else {
			Error(w, http.StatusBadRequest, "validation_error", "to must be RFC3339 format")
			return
		}
	}

	if to.Sub(from) > 24*time.Hour {
		Error(w, http.StatusBadRequest, "validation_error", "time range must not exceed 24 hours")
		return
	}

	records, err := h.locationService.GetVisibleHistory(r.Context(), claims.UserID, claims.IsAdmin, from, to)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, records)
}

// GetUserHistory handles GET /api/v1/users/{userId}/locations/history?from=&to=&device_id=
// Returns location history for a specific user.
// Admins can query any user; non-admins can only query users who share a group.
// Optional device_id param filters to a single device.
func (h *LocationHandler) GetUserHistory(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	targetUserID, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}

	now := time.Now()
	from := now.Add(-1 * time.Hour)
	to := now

	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		} else {
			Error(w, http.StatusBadRequest, "validation_error", "from must be RFC3339 format")
			return
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		} else {
			Error(w, http.StatusBadRequest, "validation_error", "to must be RFC3339 format")
			return
		}
	}

	if to.Sub(from) > 24*time.Hour {
		Error(w, http.StatusBadRequest, "validation_error", "time range must not exceed 24 hours")
		return
	}

	deviceID, err := parseOptionalDeviceID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	records, err := h.locationService.GetUserHistory(r.Context(), targetUserID, claims.UserID, claims.IsAdmin, from, to, deviceID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, records)
}

// GetAllLocations handles GET /api/v1/locations (admin only)
// Returns the latest location for every device across all groups.
func (h *LocationHandler) GetAllLocations(w http.ResponseWriter, r *http.Request) {
	records, err := h.locationService.GetAllLatest(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, records)
}
