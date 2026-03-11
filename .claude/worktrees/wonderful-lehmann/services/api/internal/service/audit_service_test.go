package service

import (
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/model"
	mockrepo "github.com/vincenty/api/internal/repository/mock"
)

// newTestPermSvc creates a PermissionPolicyService backed by a mock that
// returns no stored policy (so DefaultPermissionPolicy is used).
func newTestPermSvc() *PermissionPolicyService {
	settingsRepo := &mockrepo.ServerSettingsRepo{
		GetFn: func(ctx context.Context, key string) (*model.ServerSetting, error) {
			return nil, model.ErrNotFound("setting")
		},
	}
	return NewPermissionPolicyService(settingsRepo)
}

func TestAuditService_LogAction(t *testing.T) {
	var created *model.AuditLog
	auditRepo := &mockrepo.AuditRepo{
		CreateFn: func(ctx context.Context, log *model.AuditLog) error {
			created = log
			return nil
		},
	}
	svc := NewAuditService(auditRepo, nil, nil)

	userID := uuid.New()
	resID := uuid.New()
	err := svc.LogAction(context.Background(), CreateAuditParams{
		UserID:       userID,
		Action:       "user.create",
		ResourceType: "user",
		ResourceID:   &resID,
		IPAddress:    "1.2.3.4",
	})

	if err != nil {
		t.Fatalf("LogAction() error = %v", err)
	}
	if created == nil {
		t.Fatal("expected audit log to be created")
	}
	if created.UserID != userID {
		t.Errorf("UserID = %v, want %v", created.UserID, userID)
	}
	if created.Action != "user.create" {
		t.Errorf("Action = %q, want %q", created.Action, "user.create")
	}
	if created.ResourceID == nil || *created.ResourceID != resID {
		t.Errorf("ResourceID = %v, want %v", created.ResourceID, resID)
	}
	if created.IPAddress != "1.2.3.4" {
		t.Errorf("IPAddress = %q, want %q", created.IPAddress, "1.2.3.4")
	}
	if created.ID == uuid.Nil {
		t.Error("expected non-nil UUID for log ID")
	}
}

func TestAuditService_GetMyLogs(t *testing.T) {
	userID := uuid.New()
	expected := []model.AuditLogWithUser{{AuditLog: model.AuditLog{Action: "test"}, Username: "bob"}}

	auditRepo := &mockrepo.AuditRepo{
		ListByUserFn: func(ctx context.Context, uid uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
			if uid != userID {
				t.Errorf("ListByUser called with %v, want %v", uid, userID)
			}
			return expected, 1, nil
		},
	}
	svc := NewAuditService(auditRepo, nil, nil)

	logs, total, err := svc.GetMyLogs(context.Background(), userID, model.AuditFilters{})
	if err != nil {
		t.Fatalf("GetMyLogs() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(logs) != 1 || logs[0].Action != "test" {
		t.Errorf("unexpected logs: %v", logs)
	}
}

func TestAuditService_GetGroupLogs_SystemAdmin(t *testing.T) {
	groupID := uuid.New()
	callerID := uuid.New()
	expected := []model.AuditLogWithUser{{AuditLog: model.AuditLog{Action: "group.create"}}}

	auditRepo := &mockrepo.AuditRepo{
		ListByGroupFn: func(ctx context.Context, gid uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
			return expected, 1, nil
		},
	}
	svc := NewAuditService(auditRepo, nil, nil)

	logs, total, err := svc.GetGroupLogs(context.Background(), groupID, callerID, true, model.AuditFilters{})
	if err != nil {
		t.Fatalf("GetGroupLogs() error = %v", err)
	}
	if total != 1 || len(logs) != 1 {
		t.Errorf("unexpected result: logs=%d, total=%d", len(logs), total)
	}
}

func TestAuditService_GetGroupLogs_GroupAdmin(t *testing.T) {
	groupID := uuid.New()
	callerID := uuid.New()

	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{IsGroupAdmin: true}, nil
		},
	}
	auditRepo := &mockrepo.AuditRepo{
		ListByGroupFn: func(ctx context.Context, gid uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
			return nil, 0, nil
		},
	}
	svc := NewAuditService(auditRepo, groupRepo, newTestPermSvc())

	_, _, err := svc.GetGroupLogs(context.Background(), groupID, callerID, false, model.AuditFilters{})
	if err != nil {
		t.Fatalf("GetGroupLogs() error = %v", err)
	}
}

func TestAuditService_GetGroupLogs_NonAdmin_Forbidden(t *testing.T) {
	groupID := uuid.New()
	callerID := uuid.New()

	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{IsGroupAdmin: false, CanRead: true}, nil
		},
	}
	svc := NewAuditService(nil, groupRepo, newTestPermSvc())

	_, _, err := svc.GetGroupLogs(context.Background(), groupID, callerID, false, model.AuditFilters{})
	if err == nil {
		t.Fatal("expected forbidden error")
	}
}

func TestAuditService_GetGroupLogs_NotMember_Forbidden(t *testing.T) {
	groupID := uuid.New()
	callerID := uuid.New()

	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return nil, model.ErrNotFound("group member")
		},
	}
	svc := NewAuditService(nil, groupRepo, newTestPermSvc())

	_, _, err := svc.GetGroupLogs(context.Background(), groupID, callerID, false, model.AuditFilters{})
	if err == nil {
		t.Fatal("expected error for non-member")
	}
}

func TestAuditService_GetAllLogs(t *testing.T) {
	expected := []model.AuditLogWithUser{{AuditLog: model.AuditLog{Action: "all"}}}
	auditRepo := &mockrepo.AuditRepo{
		ListAllFn: func(ctx context.Context, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
			return expected, 5, nil
		},
	}
	svc := NewAuditService(auditRepo, nil, nil)

	logs, total, err := svc.GetAllLogs(context.Background(), model.AuditFilters{})
	if err != nil {
		t.Fatalf("GetAllLogs() error = %v", err)
	}
	if total != 5 || len(logs) != 1 {
		t.Errorf("unexpected result: logs=%d, total=%d", len(logs), total)
	}
}

func TestAuditService_ExportCSV(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	lat := 33.86
	resourceID := uuid.New()
	displayName := "Bob Smith"

	logs := []model.AuditLogWithUser{
		{
			AuditLog: model.AuditLog{
				Action:       "user.create",
				ResourceType: "user",
				ResourceID:   &resourceID,
				IPAddress:    "1.2.3.4",
				Lat:          &lat,
				CreatedAt:    now,
			},
			Username:    "bob",
			DisplayName: &displayName,
		},
	}

	svc := NewAuditService(nil, nil, nil)
	data, err := svc.ExportCSV(logs)
	if err != nil {
		t.Fatalf("ExportCSV() error = %v", err)
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(records) != 2 { // header + 1 row
		t.Fatalf("expected 2 records (header + 1 row), got %d", len(records))
	}

	// Verify header
	if records[0][0] != "Time" {
		t.Errorf("header[0] = %q, want %q", records[0][0], "Time")
	}

	// Verify data row uses display name
	if records[1][1] != "Bob Smith" {
		t.Errorf("user = %q, want %q", records[1][1], "Bob Smith")
	}
	if records[1][2] != "user.create" {
		t.Errorf("action = %q, want %q", records[1][2], "user.create")
	}
}

func TestAuditService_ExportCSV_Empty(t *testing.T) {
	svc := NewAuditService(nil, nil, nil)
	data, err := svc.ExportCSV(nil)
	if err != nil {
		t.Fatalf("ExportCSV() error = %v", err)
	}
	// Should still have header row
	if !strings.HasPrefix(string(data), "Time,") {
		t.Errorf("expected CSV header, got %q", string(data))
	}
}

func TestAuditService_ExportCSV_NoDisplayName(t *testing.T) {
	logs := []model.AuditLogWithUser{
		{
			AuditLog: model.AuditLog{
				Action:       "test",
				ResourceType: "test",
				CreatedAt:    time.Now(),
			},
			Username: "alice",
		},
	}

	svc := NewAuditService(nil, nil, nil)
	data, err := svc.ExportCSV(logs)
	if err != nil {
		t.Fatalf("ExportCSV() error = %v", err)
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	records, _ := reader.ReadAll()
	if records[1][1] != "alice" {
		t.Errorf("user = %q, want %q (fallback to username)", records[1][1], "alice")
	}
}

func TestAuditService_ExportJSON(t *testing.T) {
	logs := []model.AuditLogWithUser{
		{
			AuditLog: model.AuditLog{Action: "test", ResourceType: "user"},
			Username: "bob",
		},
	}

	svc := NewAuditService(nil, nil, nil)
	data, err := svc.ExportJSON(logs)
	if err != nil {
		t.Fatalf("ExportJSON() error = %v", err)
	}
	if !strings.Contains(string(data), `"action":"test"`) {
		t.Errorf("JSON output missing action: %s", string(data))
	}
}
