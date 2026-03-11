package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/middleware"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/service"
)

// AuditHandler handles audit log HTTP endpoints.
type AuditHandler struct {
	auditService *service.AuditService
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(auditService *service.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// GetMyLogs handles GET /api/v1/audit-logs/me
func (h *AuditHandler) GetMyLogs(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	f := parseAuditFilters(r)

	logs, total, err := h.auditService.GetMyLogs(r.Context(), claims.UserID, f)
	if err != nil {
		HandleError(w, err)
		return
	}

	writeAuditListResponse(w, logs, total, f)
}

// GetGroupLogs handles GET /api/v1/groups/{id}/audit-logs
func (h *AuditHandler) GetGroupLogs(w http.ResponseWriter, r *http.Request) {
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

	f := parseAuditFilters(r)

	logs, total, err := h.auditService.GetGroupLogs(r.Context(), groupID, claims.UserID, claims.IsAdmin, f)
	if err != nil {
		HandleError(w, err)
		return
	}

	writeAuditListResponse(w, logs, total, f)
}

// GetAllLogs handles GET /api/v1/audit-logs (admin only)
func (h *AuditHandler) GetAllLogs(w http.ResponseWriter, r *http.Request) {
	f := parseAuditFilters(r)

	logs, total, err := h.auditService.GetAllLogs(r.Context(), f)
	if err != nil {
		HandleError(w, err)
		return
	}

	writeAuditListResponse(w, logs, total, f)
}

// ExportMyLogs handles GET /api/v1/audit-logs/me/export?format=csv|json
func (h *AuditHandler) ExportMyLogs(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	f := parseAuditFilters(r)
	// For export, remove pagination limits — fetch all matching records
	f.Page = 1
	f.PageSize = 10000

	logs, _, err := h.auditService.GetMyLogs(r.Context(), claims.UserID, f)
	if err != nil {
		HandleError(w, err)
		return
	}

	writeAuditExport(w, r, h.auditService, logs)
}

// ExportAllLogs handles GET /api/v1/audit-logs/export?format=csv|json (admin only)
func (h *AuditHandler) ExportAllLogs(w http.ResponseWriter, r *http.Request) {
	f := parseAuditFilters(r)
	f.Page = 1
	f.PageSize = 10000

	logs, _, err := h.auditService.GetAllLogs(r.Context(), f)
	if err != nil {
		HandleError(w, err)
		return
	}

	writeAuditExport(w, r, h.auditService, logs)
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func parseAuditFilters(r *http.Request) model.AuditFilters {
	page, pageSize := PaginationParams(r)
	f := model.AuditFilters{
		Action:       r.URL.Query().Get("action"),
		ResourceType: r.URL.Query().Get("resource_type"),
		Page:         page,
		PageSize:     pageSize,
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

func writeAuditListResponse(w http.ResponseWriter, logs []model.AuditLogWithUser, total int, f model.AuditFilters) {
	resp := make([]model.AuditLogResponse, len(logs))
	for i := range logs {
		resp[i] = logs[i].ToResponse()
	}
	JSON(w, http.StatusOK, model.ListResponse[model.AuditLogResponse]{
		Data:     resp,
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
	})
}

func writeAuditExport(w http.ResponseWriter, r *http.Request, svc *service.AuditService, logs []model.AuditLogWithUser) {
	format := r.URL.Query().Get("format")
	ts := time.Now().UTC().Format("20060102T150405Z")

	switch format {
	case "csv":
		data, err := svc.ExportCSV(logs)
		if err != nil {
			Error(w, http.StatusInternalServerError, "internal_error", "failed to generate CSV")
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="audit_logs_%s.csv"`, ts))
		w.Write(data)

	default: // json
		data, err := svc.ExportJSON(logs)
		if err != nil {
			Error(w, http.StatusInternalServerError, "internal_error", "failed to generate JSON")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="audit_logs_%s.json"`, ts))
		w.Write(data)
	}
}
