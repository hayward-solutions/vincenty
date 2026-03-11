package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/middleware"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/service"
)

// GroupHandler handles group management endpoints.
type GroupHandler struct {
	groupService *service.GroupService
}

// NewGroupHandler creates a new GroupHandler.
func NewGroupHandler(groupService *service.GroupService) *GroupHandler {
	return &GroupHandler{groupService: groupService}
}

// --------------------------------------------------------------------------
// Group CRUD (admin)
// --------------------------------------------------------------------------

// List handles GET /api/v1/groups (admin: all groups)
func (h *GroupHandler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize := PaginationParams(r)

	groups, counts, total, err := h.groupService.List(r.Context(), page, pageSize)
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.GroupResponse, len(groups))
	for i, g := range groups {
		items[i] = g.ToResponse(counts[i])
	}

	JSON(w, http.StatusOK, model.ListResponse[model.GroupResponse]{
		Data:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Create handles POST /api/v1/groups (admin)
func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[model.CreateGroupRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	group, count, err := h.groupService.Create(r.Context(), &req, claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, group.ToResponse(count))
}

// Get handles GET /api/v1/groups/{id} (admin)
func (h *GroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid group id")
		return
	}

	group, count, err := h.groupService.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, group.ToResponse(count))
}

// Update handles PUT /api/v1/groups/{id} (admin)
func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid group id")
		return
	}

	req, err := Decode[model.UpdateGroupRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	group, count, err := h.groupService.Update(r.Context(), id, &req)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, group.ToResponse(count))
}

// Delete handles DELETE /api/v1/groups/{id} (admin)
func (h *GroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid group id")
		return
	}

	if err := h.groupService.Delete(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateMarker handles PUT /api/v1/groups/{id}/marker (group admin or system admin)
func (h *GroupHandler) UpdateMarker(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid group id")
		return
	}

	req, err := Decode[model.UpdateGroupMarkerRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	group, count, err := h.groupService.UpdateMarker(r.Context(), id, &req, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, group.ToResponse(count))
}

// --------------------------------------------------------------------------
// My Groups (authenticated user)
// --------------------------------------------------------------------------

// ListMyGroups handles GET /api/v1/users/me/groups (authenticated)
func (h *GroupHandler) ListMyGroups(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	groups, counts, err := h.groupService.ListByUserID(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.GroupResponse, len(groups))
	for i, g := range groups {
		items[i] = g.ToResponse(counts[i])
	}

	JSON(w, http.StatusOK, items)
}

// --------------------------------------------------------------------------
// Group Members
// --------------------------------------------------------------------------

// ListMembers handles GET /api/v1/groups/{id}/members (authenticated)
func (h *GroupHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
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

	members, err := h.groupService.ListMembers(r.Context(), groupID, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	items := make([]model.GroupMemberResponse, len(members))
	for i, m := range members {
		items[i] = m.ToResponse()
	}

	JSON(w, http.StatusOK, items)
}

// AddMember handles POST /api/v1/groups/{id}/members (admin or group admin)
func (h *GroupHandler) AddMember(w http.ResponseWriter, r *http.Request) {
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

	req, err := Decode[model.AddGroupMemberRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	member, err := h.groupService.AddMember(r.Context(), groupID, &req, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, member.ToResponse())
}

// UpdateMember handles PUT /api/v1/groups/{id}/members/{userId} (admin or group admin)
func (h *GroupHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
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

	memberUserID, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}

	req, err := Decode[model.UpdateGroupMemberRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	member, err := h.groupService.UpdateMember(r.Context(), groupID, memberUserID, &req, claims.UserID, claims.IsAdmin)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, member.ToResponse())
}

// RemoveMember handles DELETE /api/v1/groups/{id}/members/{userId} (admin or group admin)
func (h *GroupHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
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

	memberUserID, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}

	if err := h.groupService.RemoveMember(r.Context(), groupID, memberUserID, claims.UserID, claims.IsAdmin); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
