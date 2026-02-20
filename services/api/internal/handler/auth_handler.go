package handler

import (
	"net/http"

	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[model.LoginRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	resp, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// Refresh handles POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[model.RefreshRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	resp, err := h.authService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[model.LogoutRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	if req.RefreshToken != "" {
		_ = h.authService.Logout(r.Context(), req.RefreshToken)
	}

	JSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}
