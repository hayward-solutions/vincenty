package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// GroupService handles group management business logic.
type GroupService struct {
	groupRepo *repository.GroupRepository
	userRepo  *repository.UserRepository
}

// NewGroupService creates a new GroupService.
func NewGroupService(groupRepo *repository.GroupRepository, userRepo *repository.UserRepository) *GroupService {
	return &GroupService{groupRepo: groupRepo, userRepo: userRepo}
}

// --------------------------------------------------------------------------
// Group CRUD (admin only)
// --------------------------------------------------------------------------

// Create creates a new group. createdBy is the admin user's ID.
func (s *GroupService) Create(ctx context.Context, req *model.CreateGroupRequest, createdBy uuid.UUID) (*model.Group, int, error) {
	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}

	group := &model.Group{
		Name:        req.Name,
		Description: desc,
		CreatedBy:   &createdBy,
	}

	if req.MarkerIcon != nil {
		group.MarkerIcon = *req.MarkerIcon
	}
	if req.MarkerColor != nil {
		group.MarkerColor = *req.MarkerColor
	}

	if err := s.groupRepo.Create(ctx, group); err != nil {
		return nil, 0, err
	}

	return group, 0, nil
}

// GetByID retrieves a group by ID, including its member count.
func (s *GroupService) GetByID(ctx context.Context, id uuid.UUID) (*model.Group, int, error) {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.groupRepo.MemberCount(ctx, id)
	if err != nil {
		return nil, 0, err
	}

	return group, count, nil
}

// List retrieves a paginated list of all groups (admin).
func (s *GroupService) List(ctx context.Context, page, pageSize int) ([]model.Group, []int, int, error) {
	return s.groupRepo.List(ctx, page, pageSize)
}

// ListByUserID retrieves groups that a user is a member of.
func (s *GroupService) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Group, []int, error) {
	return s.groupRepo.ListByUserID(ctx, userID)
}

// Update modifies a group (admin only).
func (s *GroupService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateGroupRequest) (*model.Group, int, error) {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return nil, 0, err
	}

	if req.Name != nil {
		if *req.Name == "" {
			return nil, 0, model.ErrValidation("name cannot be empty")
		}
		if len(*req.Name) > 255 {
			return nil, 0, model.ErrValidation("name must be 255 characters or less")
		}
		group.Name = *req.Name
	}

	if req.Description != nil {
		group.Description = req.Description
	}

	if req.MarkerIcon != nil {
		if !model.AllowedMarkerIcons[*req.MarkerIcon] {
			return nil, 0, model.ErrValidation("invalid marker_icon value")
		}
		group.MarkerIcon = *req.MarkerIcon
	}

	if req.MarkerColor != nil {
		if !model.HexColorRegex.MatchString(*req.MarkerColor) {
			return nil, 0, model.ErrValidation("marker_color must be a valid hex color (e.g. #ff0000)")
		}
		group.MarkerColor = *req.MarkerColor
	}

	if err := s.groupRepo.Update(ctx, group); err != nil {
		return nil, 0, err
	}

	count, err := s.groupRepo.MemberCount(ctx, id)
	if err != nil {
		return nil, 0, err
	}

	return group, count, nil
}

// UpdateMarker updates a group's marker icon and color.
// Accessible by group admins and system admins.
func (s *GroupService) UpdateMarker(ctx context.Context, groupID uuid.UUID, req *model.UpdateGroupMarkerRequest, callerID uuid.UUID, callerIsAdmin bool) (*model.Group, int, error) {
	// Verify the group exists
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, 0, err
	}

	// Check permission: must be system admin or group admin
	if !callerIsAdmin {
		if err := s.requireGroupAdmin(ctx, groupID, callerID); err != nil {
			return nil, 0, err
		}
	}

	// Apply changes
	icon := group.MarkerIcon
	color := group.MarkerColor
	if req.MarkerIcon != nil {
		icon = *req.MarkerIcon
	}
	if req.MarkerColor != nil {
		color = *req.MarkerColor
	}

	updated, err := s.groupRepo.UpdateMarker(ctx, groupID, icon, color)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.groupRepo.MemberCount(ctx, groupID)
	if err != nil {
		return nil, 0, err
	}

	return updated, count, nil
}

// Delete removes a group (admin only).
func (s *GroupService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.groupRepo.Delete(ctx, id)
}

// --------------------------------------------------------------------------
// Group Members
// --------------------------------------------------------------------------

// AddMember adds a user to a group.
// callerID + callerIsAdmin are used for permission checks.
// System admins can always add members; group admins can add to their own groups.
func (s *GroupService) AddMember(ctx context.Context, groupID uuid.UUID, req *model.AddGroupMemberRequest, callerID uuid.UUID, callerIsAdmin bool) (*model.GroupMemberWithUser, error) {
	// Verify the group exists
	if _, err := s.groupRepo.GetByID(ctx, groupID); err != nil {
		return nil, err
	}

	// Check permission: must be system admin or group admin
	if !callerIsAdmin {
		if err := s.requireGroupAdmin(ctx, groupID, callerID); err != nil {
			return nil, err
		}
	}

	userID, _ := uuid.Parse(req.UserID) // already validated

	// Verify the target user exists
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, err
	}

	canRead := true
	if req.CanRead != nil {
		canRead = *req.CanRead
	}
	canWrite := false
	if req.CanWrite != nil {
		canWrite = *req.CanWrite
	}
	isGroupAdmin := false
	if req.IsGroupAdmin != nil {
		// Only system admins can grant group admin
		if *req.IsGroupAdmin && !callerIsAdmin {
			return nil, model.ErrForbidden("only system admins can grant group admin role")
		}
		isGroupAdmin = *req.IsGroupAdmin
	}

	member := &model.GroupMember{
		GroupID:      groupID,
		UserID:       userID,
		CanRead:      canRead,
		CanWrite:     canWrite,
		IsGroupAdmin: isGroupAdmin,
	}

	if err := s.groupRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	// Fetch the user details for the response
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &model.GroupMemberWithUser{
		GroupMember: *member,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	}, nil
}

// ListMembers retrieves all members of a group.
// System admins and group members with read access can view members.
func (s *GroupService) ListMembers(ctx context.Context, groupID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) ([]model.GroupMemberWithUser, error) {
	// Verify the group exists
	if _, err := s.groupRepo.GetByID(ctx, groupID); err != nil {
		return nil, err
	}

	// Check permission: must be system admin or member of the group
	if !callerIsAdmin {
		if _, err := s.groupRepo.GetMember(ctx, groupID, callerID); err != nil {
			return nil, model.ErrForbidden("you are not a member of this group")
		}
	}

	return s.groupRepo.ListMembers(ctx, groupID)
}

// UpdateMember modifies a member's permissions.
// System admins can always update; group admins can update non-admin members in their groups.
func (s *GroupService) UpdateMember(ctx context.Context, groupID, memberUserID uuid.UUID, req *model.UpdateGroupMemberRequest, callerID uuid.UUID, callerIsAdmin bool) (*model.GroupMemberWithUser, error) {
	// Get the existing membership
	member, err := s.groupRepo.GetMember(ctx, groupID, memberUserID)
	if err != nil {
		return nil, err
	}

	// Check permission
	if !callerIsAdmin {
		if err := s.requireGroupAdmin(ctx, groupID, callerID); err != nil {
			return nil, err
		}
		// Group admins cannot modify other group admins
		if member.IsGroupAdmin && memberUserID != callerID {
			return nil, model.ErrForbidden("group admins cannot modify other group admins")
		}
	}

	if req.CanRead != nil {
		member.CanRead = *req.CanRead
	}
	if req.CanWrite != nil {
		member.CanWrite = *req.CanWrite
	}
	if req.IsGroupAdmin != nil {
		// Only system admins can grant/revoke group admin
		if !callerIsAdmin {
			return nil, model.ErrForbidden("only system admins can change group admin role")
		}
		member.IsGroupAdmin = *req.IsGroupAdmin
	}

	if err := s.groupRepo.UpdateMember(ctx, member); err != nil {
		return nil, err
	}

	// Fetch user details for response
	user, err := s.userRepo.GetByID(ctx, memberUserID)
	if err != nil {
		return nil, err
	}

	return &model.GroupMemberWithUser{
		GroupMember: *member,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	}, nil
}

// RemoveMember removes a user from a group.
// System admins can always remove; group admins can remove non-admin members.
func (s *GroupService) RemoveMember(ctx context.Context, groupID, memberUserID uuid.UUID, callerID uuid.UUID, callerIsAdmin bool) error {
	// Verify group exists
	if _, err := s.groupRepo.GetByID(ctx, groupID); err != nil {
		return err
	}

	// Check permission
	if !callerIsAdmin {
		if err := s.requireGroupAdmin(ctx, groupID, callerID); err != nil {
			return err
		}
		// Group admins cannot remove other group admins
		targetMember, err := s.groupRepo.GetMember(ctx, groupID, memberUserID)
		if err != nil {
			return err
		}
		if targetMember.IsGroupAdmin {
			return model.ErrForbidden("group admins cannot remove other group admins")
		}
	}

	return s.groupRepo.RemoveMember(ctx, groupID, memberUserID)
}

// requireGroupAdmin checks that the caller is a group admin for the given group.
func (s *GroupService) requireGroupAdmin(ctx context.Context, groupID, userID uuid.UUID) error {
	member, err := s.groupRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		return model.ErrForbidden("you do not have permission to manage this group")
	}
	if !member.IsGroupAdmin {
		return model.ErrForbidden("you must be a group admin to perform this action")
	}
	return nil
}
