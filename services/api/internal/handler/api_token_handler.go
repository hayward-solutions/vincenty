package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/service"
)

// APITokenHandler handles HTTP requests for API token management.
type APITokenHandler struct {
	svc *service.APITokenService
}

// NewAPITokenHandler creates a new APITokenHandler.
func NewAPITokenHandler(svc *service.APITokenService) *APITokenHandler {
	return &APITokenHandler{svc: svc}
}

// Create handles POST /api/v1/users/me/api-tokens.
func (h *APITokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}

	req, err := Decode[model.CreateAPITokenRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp, err := h.svc.Create(r.Context(), claims.UserID, &req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, resp)
}

// List handles GET /api/v1/users/me/api-tokens.
func (h *APITokenHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}

	tokens, err := h.svc.List(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, tokens)
}

// Delete handles DELETE /api/v1/users/me/api-tokens/{id}.
func (h *APITokenHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}

	tokenID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid token id")
		return
	}

	if err := h.svc.Delete(r.Context(), claims.UserID, tokenID); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
