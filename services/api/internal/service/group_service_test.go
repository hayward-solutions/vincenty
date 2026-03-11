package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/model"
	mockrepo "github.com/vincenty/api/internal/repository/mock"
)

func TestGroupService_Create(t *testing.T) {
	var created *model.Group
	groupRepo := &mockrepo.GroupRepo{
		CreateFn: func(ctx context.Context, g *model.Group) error {
			created = g
			return nil
		},
	}
	svc := NewGroupService(groupRepo, nil, nil, nil)

	callerID := uuid.New()
	icon := "shield"
	color := "#ff0000"
	req := &model.CreateGroupRequest{
		Name:        "Alpha Team",
		Description: "Test group",
		MarkerIcon:  &icon,
		MarkerColor: &color,
	}

	group, _, err := svc.Create(context.Background(), req, callerID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if group.Name != "Alpha Team" {
		t.Errorf("Name = %q, want %q", group.Name, "Alpha Team")
	}
	if created.MarkerIcon != "shield" {
		t.Errorf("MarkerIcon = %q, want %q", created.MarkerIcon, "shield")
	}
	if created.CreatedBy == nil || *created.CreatedBy != callerID {
		t.Errorf("CreatedBy = %v, want %v", created.CreatedBy, callerID)
	}
}

func TestGroupService_GetByID(t *testing.T) {
	groupID := uuid.New()
	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID, Name: "Test"}, nil
		},
		MemberCountFn: func(ctx context.Context, gid uuid.UUID) (int, error) {
			return 5, nil
		},
	}
	svc := NewGroupService(groupRepo, nil, nil, nil)

	group, count, err := svc.GetByID(context.Background(), groupID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if group.Name != "Test" {
		t.Errorf("Name = %q, want %q", group.Name, "Test")
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
}

func TestGroupService_GetByID_NotFound(t *testing.T) {
	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return nil, model.ErrNotFound("group")
		},
	}
	svc := NewGroupService(groupRepo, nil, nil, nil)

	_, _, err := svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestGroupService_Update(t *testing.T) {
	groupID := uuid.New()
	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID, Name: "Old"}, nil
		},
		UpdateFn: func(ctx context.Context, g *model.Group) error {
			return nil
		},
		MemberCountFn: func(ctx context.Context, gid uuid.UUID) (int, error) {
			return 3, nil
		},
	}
	svc := NewGroupService(groupRepo, nil, nil, nil)

	newName := "New Name"
	group, count, err := svc.Update(context.Background(), groupID, &model.UpdateGroupRequest{Name: &newName})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if group.Name != "New Name" {
		t.Errorf("Name = %q, want %q", group.Name, "New Name")
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestGroupService_Update_EmptyName(t *testing.T) {
	groupID := uuid.New()
	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID, Name: "Old"}, nil
		},
	}
	svc := NewGroupService(groupRepo, nil, nil, nil)

	emptyName := ""
	_, _, err := svc.Update(context.Background(), groupID, &model.UpdateGroupRequest{Name: &emptyName})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestGroupService_Update_LongName(t *testing.T) {
	groupID := uuid.New()
	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID, Name: "Old"}, nil
		},
	}
	svc := NewGroupService(groupRepo, nil, nil, nil)

	longName := string(make([]byte, 256))
	_, _, err := svc.Update(context.Background(), groupID, &model.UpdateGroupRequest{Name: &longName})
	if err == nil {
		t.Fatal("expected error for long name")
	}
}

func TestGroupService_Delete(t *testing.T) {
	deleted := false
	groupRepo := &mockrepo.GroupRepo{
		DeleteFn: func(ctx context.Context, id uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	svc := NewGroupService(groupRepo, nil, nil, nil)

	err := svc.Delete(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called")
	}
}

func TestGroupService_AddMember_SystemAdmin(t *testing.T) {
	groupID := uuid.New()
	userID := uuid.New()
	callerID := uuid.New()

	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
		AddMemberFn: func(ctx context.Context, m *model.GroupMember) error {
			return nil
		},
	}
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, Username: "bob"}, nil
		},
	}
	svc := NewGroupService(groupRepo, userRepo, nil, nil)

	req := &model.AddGroupMemberRequest{UserID: userID.String()}
	member, err := svc.AddMember(context.Background(), groupID, req, callerID, true)
	if err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}
	if member.Username != "bob" {
		t.Errorf("Username = %q, want %q", member.Username, "bob")
	}
}

func TestGroupService_AddMember_NonAdmin_GroupAdmin(t *testing.T) {
	groupID := uuid.New()
	userID := uuid.New()
	callerID := uuid.New()

	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			if uid == callerID {
				return &model.GroupMember{IsGroupAdmin: true}, nil
			}
			return nil, model.ErrNotFound("member")
		},
		AddMemberFn: func(ctx context.Context, m *model.GroupMember) error {
			return nil
		},
	}
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, Username: "alice"}, nil
		},
	}
	svc := NewGroupService(groupRepo, userRepo, newTestPermSvc(), nil)

	req := &model.AddGroupMemberRequest{UserID: userID.String()}
	member, err := svc.AddMember(context.Background(), groupID, req, callerID, false)
	if err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}
	if member.Username != "alice" {
		t.Errorf("Username = %q, want %q", member.Username, "alice")
	}
}

func TestGroupService_AddMember_NonAdminCannotGrantGroupAdmin(t *testing.T) {
	groupID := uuid.New()
	userID := uuid.New()
	callerID := uuid.New()

	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{IsGroupAdmin: true}, nil
		},
	}
	userRepo := &mockrepo.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID}, nil
		},
	}
	svc := NewGroupService(groupRepo, userRepo, newTestPermSvc(), nil)

	isGroupAdmin := true
	req := &model.AddGroupMemberRequest{UserID: userID.String(), IsGroupAdmin: &isGroupAdmin}
	_, err := svc.AddMember(context.Background(), groupID, req, callerID, false)
	if err == nil {
		t.Fatal("expected forbidden error when non-admin grants group admin")
	}
}

func TestGroupService_ListMembers_SystemAdmin(t *testing.T) {
	groupID := uuid.New()
	expected := []model.GroupMemberWithUser{{Username: "bob"}}

	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
		ListMembersFn: func(ctx context.Context, gid uuid.UUID) ([]model.GroupMemberWithUser, error) {
			return expected, nil
		},
	}
	svc := NewGroupService(groupRepo, nil, nil, nil)

	members, err := svc.ListMembers(context.Background(), groupID, uuid.New(), true)
	if err != nil {
		t.Fatalf("ListMembers() error = %v", err)
	}
	if len(members) != 1 {
		t.Errorf("len(members) = %d, want 1", len(members))
	}
}

func TestGroupService_ListMembers_NonMember_Forbidden(t *testing.T) {
	groupID := uuid.New()
	callerID := uuid.New()

	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return nil, model.ErrNotFound("group member")
		},
	}
	svc := NewGroupService(groupRepo, nil, nil, nil)

	_, err := svc.ListMembers(context.Background(), groupID, callerID, false)
	if err == nil {
		t.Fatal("expected forbidden error for non-member")
	}
}

func TestGroupService_RemoveMember_SystemAdmin(t *testing.T) {
	groupID := uuid.New()
	memberID := uuid.New()
	removed := false

	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
		RemoveMemberFn: func(ctx context.Context, gid, uid uuid.UUID) error {
			removed = true
			return nil
		},
	}
	svc := NewGroupService(groupRepo, nil, nil, nil)

	err := svc.RemoveMember(context.Background(), groupID, memberID, uuid.New(), true)
	if err != nil {
		t.Fatalf("RemoveMember() error = %v", err)
	}
	if !removed {
		t.Error("expected RemoveMember to be called")
	}
}

func TestGroupService_RemoveMember_GroupAdmin_CannotRemoveOtherAdmin(t *testing.T) {
	groupID := uuid.New()
	callerID := uuid.New()
	targetID := uuid.New()

	groupRepo := &mockrepo.GroupRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			// Both caller and target are group admins
			return &model.GroupMember{IsGroupAdmin: true}, nil
		},
	}
	svc := NewGroupService(groupRepo, nil, newTestPermSvc(), nil)

	err := svc.RemoveMember(context.Background(), groupID, targetID, callerID, false)
	if err == nil {
		t.Fatal("expected forbidden error when group admin tries to remove another group admin")
	}
}
