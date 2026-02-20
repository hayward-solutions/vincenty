package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/service"
)

// UserHandler handles user management endpoints.
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
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
