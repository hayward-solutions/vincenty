package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/gpx"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/service"
)

// LocationHandler handles location query endpoints.
type LocationHandler struct {
	locationService *service.LocationService
}

// NewLocationHandler creates a new LocationHandler.
func NewLocationHandler(locationService *service.LocationService) *LocationHandler {
	return &LocationHandler{locationService: locationService}
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

// GetMyHistory handles GET /api/v1/users/me/locations/history?from=&to=
// Returns the caller's own location history within a time range.
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

	records, err := h.locationService.GetMyHistory(r.Context(), claims.UserID, from, to)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, records)
}

// ExportGPX handles GET /api/v1/users/me/locations/export?from=&to=
// Returns the caller's own location history as a GPX file download.
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

	records, err := h.locationService.GetMyHistory(r.Context(), claims.UserID, from, to)
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

	name := fmt.Sprintf("SitAware Track %s", from.UTC().Format("2006-01-02"))
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

// GetAllLocations handles GET /api/v1/locations (admin only)
// Returns the latest location for every user across all groups.
func (h *LocationHandler) GetAllLocations(w http.ResponseWriter, r *http.Request) {
	records, err := h.locationService.GetAllLatest(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, records)
}
