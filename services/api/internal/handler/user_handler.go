package handler

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/middleware"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/service"
	"github.com/vincenty/api/internal/storage"
)

const (
	maxAvatarSize = 5 << 20 // 5 MB
)

// allowedAvatarTypes maps MIME types to file extensions for avatar uploads.
var allowedAvatarTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// UserHandler handles user management endpoints.
type UserHandler struct {
	userService *service.UserService
	storageSvc  storage.Storage
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService *service.UserService, storageSvc storage.Storage) *UserHandler {
	return &UserHandler{userService: userService, storageSvc: storageSvc}
}

// List handles GET /api/v1/users (admin)
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize := PaginationParams(r)

	users, total, err := h.userService.List(r.Context(), page, pageSize)
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.UserResponse, len(users))
	for i, u := range users {
		items[i] = u.ToResponse()
	}

	JSON(w, http.StatusOK, model.ListResponse[model.UserResponse]{
		Data:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Create handles POST /api/v1/users (admin)
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[model.CreateUserRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	user, err := h.userService.Create(r.Context(), &req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, user.ToResponse())
}

// Get handles GET /api/v1/users/{id} (admin)
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}

	user, err := h.userService.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, user.ToResponse())
}

// Update handles PUT /api/v1/users/{id} (admin)
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}

	req, err := Decode[model.UpdateUserRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	user, err := h.userService.Update(r.Context(), id, &req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, user.ToResponse())
}

// Delete handles DELETE /api/v1/users/{id} (admin)
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}

	if err := h.userService.Delete(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetMe handles GET /api/v1/users/me (authenticated)
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	user, err := h.userService.GetByID(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, user.ToResponse())
}

// UpdateMe handles PUT /api/v1/users/me (authenticated)
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[model.UpdateMeRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	user, err := h.userService.UpdateMe(r.Context(), claims.UserID, &req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, user.ToResponse())
}

// ChangePassword handles PUT /api/v1/users/me/password (authenticated)
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[model.ChangePasswordRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	if err := h.userService.ChangePassword(r.Context(), claims.UserID, &req); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "password changed successfully"})
}

// UploadAvatar handles PUT /api/v1/users/me/avatar (authenticated)
func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	// Parse multipart form (limit to maxAvatarSize in memory)
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "avatar file is required")
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > maxAvatarSize {
		Error(w, http.StatusBadRequest, "validation_error", fmt.Sprintf("avatar must be smaller than %d MB", maxAvatarSize/(1<<20)))
		return
	}

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	// Normalize: take only the MIME type portion (strip params like charset)
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	if !allowedAvatarTypes[contentType] {
		Error(w, http.StatusBadRequest, "validation_error", "avatar must be a JPEG, PNG, or WebP image")
		return
	}

	// Build a safe filename
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		switch contentType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		}
	}
	safeFilename := "avatar" + ext

	user, err := h.userService.UploadAvatar(r.Context(), claims.UserID, file, safeFilename, contentType, header.Size)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, user.ToResponse())
}

// DeleteAvatar handles DELETE /api/v1/users/me/avatar (authenticated)
func (h *UserHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	user, err := h.userService.DeleteAvatar(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, user.ToResponse())
}

// ServeAvatar handles GET /api/v1/users/{id}/avatar (authenticated with query token)
func (h *UserHandler) ServeAvatar(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}

	key, err := h.userService.GetAvatarKey(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}
	if key == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	body, contentType, contentLength, err := h.storageSvc.Download(r.Context(), key)
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal_error", "failed to retrieve avatar")
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", contentType)
	if contentLength > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", contentLength))
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, body)
}
