package handler

import (
	"net/http"

	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/service"
)

// PermissionPolicyHandler handles permission policy endpoints.
type PermissionPolicyHandler struct {
	svc *service.PermissionPolicyService
}

// NewPermissionPolicyHandler creates a new PermissionPolicyHandler.
func NewPermissionPolicyHandler(svc *service.PermissionPolicyService) *PermissionPolicyHandler {
	return &PermissionPolicyHandler{svc: svc}
}

// GetPolicy handles GET /api/v1/server/permissions (admin).
func (h *PermissionPolicyHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	policy, err := h.svc.GetPolicy(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, model.PermissionPolicyResponse{
		GroupCommunication: policy.GroupCommunication,
		GroupManagement:    policy.GroupManagement,
	})
}

// UpdatePolicy handles PUT /api/v1/server/permissions (admin).
func (h *PermissionPolicyHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[model.UpdatePermissionPolicyRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	policy := &model.PermissionPolicy{
		GroupCommunication: req.GroupCommunication,
		GroupManagement:    req.GroupManagement,
	}

	if err := h.svc.UpdatePolicy(r.Context(), policy); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, model.PermissionPolicyResponse{
		GroupCommunication: policy.GroupCommunication,
		GroupManagement:    policy.GroupManagement,
	})
}
