package model

import "fmt"

// ---------------------------------------------------------------------------
// Role identifiers — map to existing database flags
// ---------------------------------------------------------------------------

const (
	RoleServerAdmin = "server_admin" // users.is_admin
	RoleGroupAdmin  = "group_admin"  // group_members.is_group_admin
	RoleWriter      = "writer"       // group_members.can_write
	RoleReader      = "reader"       // group_members.can_read
	RoleMember      = "member"       // any row in group_members
)

// ValidCommunicationRoles are the roles allowed in the communication matrix.
var ValidCommunicationRoles = map[string]bool{
	RoleServerAdmin: true,
	RoleGroupAdmin:  true,
	RoleWriter:      true,
	RoleReader:      true,
}

// ValidManagementRoles are the roles allowed in the management matrix.
// Server admins manage via the admin panel (outside the matrix).
var ValidManagementRoles = map[string]bool{
	RoleGroupAdmin: true,
	RoleMember:     true,
}

// ---------------------------------------------------------------------------
// Action identifiers
// ---------------------------------------------------------------------------

// Group communication actions (require group membership for ALL callers).
const (
	ActionSendMessages    = "send_messages"
	ActionReadMessages    = "read_messages"
	ActionSendAttachments = "send_attachments"
	ActionShareDrawings   = "share_drawings"
	ActionShareLocation   = "share_location"
	ActionViewLocations   = "view_locations"
	ActionStartStream     = "start_stream"
	ActionViewStream      = "view_stream"
	ActionRecordStream    = "record_stream"
	ActionUsePTT          = "use_ptt"
)

// Group management actions (admin panel bypasses; matrix governs group-level roles).
const (
	ActionAddMembers    = "add_members"
	ActionRemoveMembers = "remove_members"
	ActionUpdateMembers = "update_members"
	ActionUpdateMarker  = "update_marker"
	ActionViewAuditLogs = "view_audit_logs"
)

// ValidCommunicationActions lists every recognised communication action.
var ValidCommunicationActions = map[string]bool{
	ActionSendMessages:    true,
	ActionReadMessages:    true,
	ActionSendAttachments: true,
	ActionShareDrawings:   true,
	ActionShareLocation:   true,
	ActionViewLocations:   true,
	ActionStartStream:     true,
	ActionViewStream:      true,
	ActionRecordStream:    true,
	ActionUsePTT:          true,
}

// ValidManagementActions lists every recognised management action.
var ValidManagementActions = map[string]bool{
	ActionAddMembers:    true,
	ActionRemoveMembers: true,
	ActionUpdateMembers: true,
	ActionUpdateMarker:  true,
	ActionViewAuditLogs: true,
}

// ---------------------------------------------------------------------------
// Categories
// ---------------------------------------------------------------------------

const (
	CategoryCommunication = "group_communication"
	CategoryManagement    = "group_management"
)

// ---------------------------------------------------------------------------
// PermissionPolicy — the configurable matrix
// ---------------------------------------------------------------------------

// PermissionPolicy maps actions to the roles that may perform them.
type PermissionPolicy struct {
	GroupCommunication map[string][]string `json:"group_communication"`
	GroupManagement    map[string][]string `json:"group_management"`
}

// DefaultPermissionPolicy returns the policy that matches current behaviour.
func DefaultPermissionPolicy() PermissionPolicy {
	return PermissionPolicy{
		GroupCommunication: map[string][]string{
			ActionSendMessages:    {RoleServerAdmin, RoleGroupAdmin, RoleWriter},
			ActionReadMessages:    {RoleServerAdmin, RoleGroupAdmin, RoleWriter, RoleReader},
			ActionSendAttachments: {RoleServerAdmin, RoleGroupAdmin, RoleWriter},
			ActionShareDrawings:   {RoleServerAdmin, RoleGroupAdmin, RoleWriter},
			ActionShareLocation:   {RoleServerAdmin, RoleGroupAdmin, RoleWriter, RoleReader},
			ActionViewLocations:   {RoleServerAdmin, RoleGroupAdmin, RoleWriter, RoleReader},
			ActionStartStream:     {RoleServerAdmin, RoleGroupAdmin, RoleWriter},
			ActionViewStream:      {RoleServerAdmin, RoleGroupAdmin, RoleWriter, RoleReader},
			ActionRecordStream:    {RoleServerAdmin, RoleGroupAdmin},
			ActionUsePTT:          {RoleServerAdmin, RoleGroupAdmin, RoleWriter, RoleReader},
		},
		GroupManagement: map[string][]string{
			ActionAddMembers:    {RoleGroupAdmin},
			ActionRemoveMembers: {RoleGroupAdmin},
			ActionUpdateMembers: {RoleGroupAdmin},
			ActionUpdateMarker:  {RoleGroupAdmin},
			ActionViewAuditLogs: {RoleGroupAdmin},
		},
	}
}

// CheckCommunication returns true when the caller is allowed to perform the
// given communication action in a group. The caller MUST be a group member
// (member != nil); non-members are always denied.
func (p *PermissionPolicy) CheckCommunication(action string, member *GroupMember, isServerAdmin bool) bool {
	if member == nil {
		return false
	}
	roles, ok := p.GroupCommunication[action]
	if !ok {
		return false
	}
	return matchesAny(roles, member, isServerAdmin)
}

// CheckManagement returns true when the caller is allowed to perform the
// given management action. The caller MUST be a group member (member != nil);
// server admins use the admin panel which bypasses this matrix entirely.
func (p *PermissionPolicy) CheckManagement(action string, member *GroupMember) bool {
	if member == nil {
		return false
	}
	roles, ok := p.GroupManagement[action]
	if !ok {
		return false
	}
	for _, role := range roles {
		switch role {
		case RoleGroupAdmin:
			if member.IsGroupAdmin {
				return true
			}
		case RoleMember:
			return true // any member qualifies
		}
	}
	return false
}

// matchesAny checks whether the member holds any of the given roles.
func matchesAny(roles []string, member *GroupMember, isServerAdmin bool) bool {
	for _, role := range roles {
		switch role {
		case RoleServerAdmin:
			if isServerAdmin {
				return true
			}
		case RoleGroupAdmin:
			if member.IsGroupAdmin {
				return true
			}
		case RoleWriter:
			if member.CanWrite {
				return true
			}
		case RoleReader:
			if member.CanRead {
				return true
			}
		case RoleMember:
			return true // any member qualifies
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// Validate checks that the policy contains only known actions and roles.
func (p *PermissionPolicy) Validate() error {
	for action, roles := range p.GroupCommunication {
		if !ValidCommunicationActions[action] {
			return ErrValidation(fmt.Sprintf("unknown communication action: %s", action))
		}
		for _, role := range roles {
			if !ValidCommunicationRoles[role] {
				return ErrValidation(fmt.Sprintf("invalid role %q for communication action %s", role, action))
			}
		}
	}
	for action, roles := range p.GroupManagement {
		if !ValidManagementActions[action] {
			return ErrValidation(fmt.Sprintf("unknown management action: %s", action))
		}
		for _, role := range roles {
			if !ValidManagementRoles[role] {
				return ErrValidation(fmt.Sprintf("invalid role %q for management action %s", role, action))
			}
		}
	}
	// Ensure all known actions are present
	for action := range ValidCommunicationActions {
		if _, ok := p.GroupCommunication[action]; !ok {
			return ErrValidation(fmt.Sprintf("missing communication action: %s", action))
		}
	}
	for action := range ValidManagementActions {
		if _, ok := p.GroupManagement[action]; !ok {
			return ErrValidation(fmt.Sprintf("missing management action: %s", action))
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// API request / response types
// ---------------------------------------------------------------------------

// PermissionPolicyResponse is the API response for the permission policy.
type PermissionPolicyResponse struct {
	GroupCommunication map[string][]string `json:"group_communication"`
	GroupManagement    map[string][]string `json:"group_management"`
}

// UpdatePermissionPolicyRequest is the API request to update the permission policy.
type UpdatePermissionPolicyRequest struct {
	GroupCommunication map[string][]string `json:"group_communication"`
	GroupManagement    map[string][]string `json:"group_management"`
}
