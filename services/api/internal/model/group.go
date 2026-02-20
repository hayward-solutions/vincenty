package model

import (
	"time"

	"github.com/google/uuid"
)

// Group represents a group in the system.
type Group struct {
	ID          uuid.UUID  `json:"-"`
	Name        string     `json:"-"`
	Description *string    `json:"-"`
	CreatedBy   *uuid.UUID `json:"-"`
	CreatedAt   time.Time  `json:"-"`
	UpdatedAt   time.Time  `json:"-"`
}

// GroupResponse is the JSON representation of a group returned by the API.
type GroupResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	MemberCount int        `json:"member_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ToResponse converts a Group to its API response representation.
// memberCount should be provided externally.
func (g *Group) ToResponse(memberCount int) GroupResponse {
	description := ""
	if g.Description != nil {
		description = *g.Description
	}
	return GroupResponse{
		ID:          g.ID,
		Name:        g.Name,
		Description: description,
		CreatedBy:   g.CreatedBy,
		MemberCount: memberCount,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

// CreateGroupRequest is the expected body for creating a new group.
type CreateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Validate checks that required fields are present.
func (r *CreateGroupRequest) Validate() error {
	if r.Name == "" {
		return ErrValidation("name is required")
	}
	if len(r.Name) > 255 {
		return ErrValidation("name must be 255 characters or less")
	}
	return nil
}

// UpdateGroupRequest is the expected body for updating a group.
type UpdateGroupRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// GroupMember represents a user's membership in a group.
type GroupMember struct {
	ID           uuid.UUID `json:"-"`
	GroupID      uuid.UUID `json:"-"`
	UserID       uuid.UUID `json:"-"`
	CanRead      bool      `json:"-"`
	CanWrite     bool      `json:"-"`
	IsGroupAdmin bool      `json:"-"`
	CreatedAt    time.Time `json:"-"`
	UpdatedAt    time.Time `json:"-"`
}

// GroupMemberResponse is the JSON representation of a group member.
type GroupMemberResponse struct {
	ID           uuid.UUID `json:"id"`
	GroupID      uuid.UUID `json:"group_id"`
	UserID       uuid.UUID `json:"user_id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	CanRead      bool      `json:"can_read"`
	CanWrite     bool      `json:"can_write"`
	IsGroupAdmin bool      `json:"is_group_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GroupMemberWithUser is a join result containing member + user details.
type GroupMemberWithUser struct {
	GroupMember
	Username    string
	DisplayName *string
}

// ToResponse converts a GroupMemberWithUser to its API response.
func (m *GroupMemberWithUser) ToResponse() GroupMemberResponse {
	displayName := ""
	if m.DisplayName != nil {
		displayName = *m.DisplayName
	}
	return GroupMemberResponse{
		ID:           m.ID,
		GroupID:      m.GroupID,
		UserID:       m.UserID,
		Username:     m.Username,
		DisplayName:  displayName,
		CanRead:      m.CanRead,
		CanWrite:     m.CanWrite,
		IsGroupAdmin: m.IsGroupAdmin,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// AddGroupMemberRequest is the expected body for adding a member to a group.
type AddGroupMemberRequest struct {
	UserID       string `json:"user_id"`
	CanRead      *bool  `json:"can_read"`
	CanWrite     *bool  `json:"can_write"`
	IsGroupAdmin *bool  `json:"is_group_admin"`
}

// Validate checks that required fields are present.
func (r *AddGroupMemberRequest) Validate() error {
	if r.UserID == "" {
		return ErrValidation("user_id is required")
	}
	if _, err := uuid.Parse(r.UserID); err != nil {
		return ErrValidation("user_id must be a valid UUID")
	}
	return nil
}

// UpdateGroupMemberRequest is the expected body for updating a member's permissions.
type UpdateGroupMemberRequest struct {
	CanRead      *bool `json:"can_read"`
	CanWrite     *bool `json:"can_write"`
	IsGroupAdmin *bool `json:"is_group_admin"`
}
