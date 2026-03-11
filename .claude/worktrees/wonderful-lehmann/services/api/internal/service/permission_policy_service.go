package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/repository"
)

const permissionPolicyKey = "permission_policy"

// PermissionPolicyService manages loading, caching, and checking the
// server-wide permission policy.
type PermissionPolicyService struct {
	repo repository.ServerSettingsRepo

	mu     sync.RWMutex
	cached *model.PermissionPolicy
}

// NewPermissionPolicyService creates a new PermissionPolicyService.
func NewPermissionPolicyService(repo repository.ServerSettingsRepo) *PermissionPolicyService {
	return &PermissionPolicyService{repo: repo}
}

// GetPolicy returns the current permission policy. If none is stored in the
// database the default policy is returned.
func (s *PermissionPolicyService) GetPolicy(ctx context.Context) (*model.PermissionPolicy, error) {
	s.mu.RLock()
	if s.cached != nil {
		defer s.mu.RUnlock()
		return s.cached, nil
	}
	s.mu.RUnlock()

	return s.loadPolicy(ctx)
}

// UpdatePolicy validates and persists a new permission policy, then
// invalidates the in-memory cache.
func (s *PermissionPolicyService) UpdatePolicy(ctx context.Context, policy *model.PermissionPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}

	data, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	if err := s.repo.Set(ctx, permissionPolicyKey, string(data)); err != nil {
		return err
	}

	s.mu.Lock()
	s.cached = policy
	s.mu.Unlock()

	slog.Info("permission policy updated")
	return nil
}

// CheckCommunication checks whether a group member is allowed to perform
// a communication action. The caller must be a group member (member != nil).
func (s *PermissionPolicyService) CheckCommunication(ctx context.Context, action string, member *model.GroupMember, isServerAdmin bool) (bool, error) {
	policy, err := s.GetPolicy(ctx)
	if err != nil {
		return false, err
	}
	return policy.CheckCommunication(action, member, isServerAdmin), nil
}

// RequireCommunication is like CheckCommunication but returns a ForbiddenError
// when the action is not allowed.
func (s *PermissionPolicyService) RequireCommunication(ctx context.Context, action string, member *model.GroupMember, isServerAdmin bool) error {
	allowed, err := s.CheckCommunication(ctx, action, member, isServerAdmin)
	if err != nil {
		return err
	}
	if !allowed {
		return model.ErrForbidden("you do not have permission to perform this action")
	}
	return nil
}

// CheckManagement checks whether a group member is allowed to perform
// a management action. Server admins use the admin panel and bypass this
// matrix; only group-level roles are evaluated here.
func (s *PermissionPolicyService) CheckManagement(ctx context.Context, action string, member *model.GroupMember) (bool, error) {
	policy, err := s.GetPolicy(ctx)
	if err != nil {
		return false, err
	}
	return policy.CheckManagement(action, member), nil
}

// RequireManagement is like CheckManagement but returns a ForbiddenError
// when the action is not allowed.
func (s *PermissionPolicyService) RequireManagement(ctx context.Context, action string, member *model.GroupMember) error {
	allowed, err := s.CheckManagement(ctx, action, member)
	if err != nil {
		return err
	}
	if !allowed {
		return model.ErrForbidden("you do not have permission to perform this action")
	}
	return nil
}

// loadPolicy reads the policy from the database and caches it.
func (s *PermissionPolicyService) loadPolicy(ctx context.Context) (*model.PermissionPolicy, error) {
	setting, err := s.repo.Get(ctx, permissionPolicyKey)
	if err != nil {
		// Not found → return default
		def := model.DefaultPermissionPolicy()
		s.mu.Lock()
		s.cached = &def
		s.mu.Unlock()
		return &def, nil
	}

	var policy model.PermissionPolicy
	if err := json.Unmarshal([]byte(setting.Value), &policy); err != nil {
		slog.Warn("invalid permission policy in database, using default", "error", err)
		def := model.DefaultPermissionPolicy()
		s.mu.Lock()
		s.cached = &def
		s.mu.Unlock()
		return &def, nil
	}

	s.mu.Lock()
	s.cached = &policy
	s.mu.Unlock()

	return &policy, nil
}
