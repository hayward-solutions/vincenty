package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/service"
)

// DrawingHandler handles drawing HTTP endpoints.
type DrawingHandler struct {
	drawingService *service.DrawingService
}

// NewDrawingHandler creates a new DrawingHandler.
func NewDrawingHandler(drawingService *service.DrawingService) *DrawingHandler {
	return &DrawingHandler{drawingService: drawingService}
}

// Create handles POST /api/v1/drawings
func (h *DrawingHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[service.CreateDrawingRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	dwu, err := h.drawingService.Create(r.Context(), claims.UserID, req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, dwu.ToResponse())
}

// Get handles GET /api/v1/drawings/{id}
func (h *DrawingHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	drawingID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid drawing id")
		return
	}

	dwu, err := h.drawingService.Get(r.Context(), drawingID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, dwu.ToResponse())
}

// ListOwn handles GET /api/v1/drawings
func (h *DrawingHandler) ListOwn(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	drawings, err := h.drawingService.ListOwn(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp := make([]interface{}, len(drawings))
	for i := range drawings {
		r := drawings[i].ToResponse()
		resp[i] = r
	}

	JSON(w, http.StatusOK, resp)
}

// ListShared handles GET /api/v1/drawings/shared
func (h *DrawingHandler) ListShared(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	drawings, err := h.drawingService.ListShared(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp := make([]interface{}, len(drawings))
	for i := range drawings {
		r := drawings[i].ToResponse()
		resp[i] = r
	}

	JSON(w, http.StatusOK, resp)
}

// Update handles PUT /api/v1/drawings/{id}
func (h *DrawingHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	drawingID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid drawing id")
		return
	}

	req, err := Decode[service.UpdateDrawingRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	dwu, err := h.drawingService.Update(r.Context(), drawingID, claims.UserID, claims.IsAdmin, req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, dwu.ToResponse())
}

// Delete handles DELETE /api/v1/drawings/{id}
func (h *DrawingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	drawingID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid drawing id")
		return
	}

	if err := h.drawingService.Delete(r.Context(), drawingID, claims.UserID, claims.IsAdmin); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListShares handles GET /api/v1/drawings/{id}/shares
func (h *DrawingHandler) ListShares(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	drawingID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid drawing id")
		return
	}

	shares, err := h.drawingService.ListShares(r.Context(), drawingID, claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, shares)
}

// Unshare handles DELETE /api/v1/drawings/{id}/shares/{messageId}
func (h *DrawingHandler) Unshare(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	drawingID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid drawing id")
		return
	}

	messageID, err := uuid.Parse(r.PathValue("messageId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid message id")
		return
	}

	if err := h.drawingService.Unshare(r.Context(), drawingID, claims.UserID, messageID); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Share handles POST /api/v1/drawings/{id}/share
func (h *DrawingHandler) Share(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	drawingID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid drawing id")
		return
	}

	req, err := Decode[service.ShareDrawingRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	msg, err := h.drawingService.Share(r.Context(), drawingID, claims.UserID, claims.IsAdmin, req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, msg.ToResponse())
}
