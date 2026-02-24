package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// CreateAuditParams holds the parameters for creating an audit log entry.
type CreateAuditParams struct {
	UserID       uuid.UUID
	DeviceID     *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	GroupID      *uuid.UUID
	Metadata     *json.RawMessage
	Lat          *float64
	Lng          *float64
	IPAddress    string
}

// AuditService handles audit log business logic.
type AuditService struct {
	auditRepo repository.AuditRepo
	groupRepo repository.GroupRepo
}

// NewAuditService creates a new AuditService.
func NewAuditService(auditRepo repository.AuditRepo, groupRepo repository.GroupRepo) *AuditService {
	return &AuditService{auditRepo: auditRepo, groupRepo: groupRepo}
}

// LogAction creates an audit log entry.
func (s *AuditService) LogAction(ctx context.Context, p CreateAuditParams) error {
	log := &model.AuditLog{
		ID:           uuid.New(),
		UserID:       p.UserID,
		DeviceID:     p.DeviceID,
		Action:       p.Action,
		ResourceType: p.ResourceType,
		ResourceID:   p.ResourceID,
		GroupID:      p.GroupID,
		Metadata:     p.Metadata,
		Lat:          p.Lat,
		Lng:          p.Lng,
		IPAddress:    p.IPAddress,
		CreatedAt:    time.Now(),
	}
	return s.auditRepo.Create(ctx, log)
}

// GetMyLogs returns audit logs for the calling user.
func (s *AuditService) GetMyLogs(ctx context.Context, callerID uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
	return s.auditRepo.ListByUser(ctx, callerID, f)
}

// GetGroupLogs returns audit logs scoped to a group.
// Caller must be a group admin or system admin.
func (s *AuditService) GetGroupLogs(ctx context.Context, groupID, callerID uuid.UUID, callerIsAdmin bool, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
	if !callerIsAdmin {
		member, err := s.groupRepo.GetMember(ctx, groupID, callerID)
		if err != nil {
			return nil, 0, model.ErrForbidden("you are not a member of this group")
		}
		if !member.IsGroupAdmin {
			return nil, 0, model.ErrForbidden("group admin access required")
		}
	}
	return s.auditRepo.ListByGroup(ctx, groupID, f)
}

// GetAllLogs returns all audit logs (admin only).
func (s *AuditService) GetAllLogs(ctx context.Context, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
	return s.auditRepo.ListAll(ctx, f)
}

// ExportCSV serializes audit log entries as CSV bytes.
func (s *AuditService) ExportCSV(logs []model.AuditLogWithUser) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Header
	if err := w.Write([]string{"Time", "User", "Action", "Resource Type", "Resource ID", "Group ID", "IP Address", "Lat", "Lng"}); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}

	for _, l := range logs {
		resourceID := ""
		if l.ResourceID != nil {
			resourceID = l.ResourceID.String()
		}
		groupID := ""
		if l.GroupID != nil {
			groupID = l.GroupID.String()
		}
		lat, lng := "", ""
		if l.Lat != nil {
			lat = fmt.Sprintf("%.6f", *l.Lat)
		}
		if l.Lng != nil {
			lng = fmt.Sprintf("%.6f", *l.Lng)
		}
		dn := l.Username
		if l.DisplayName != nil && *l.DisplayName != "" {
			dn = *l.DisplayName
		}

		if err := w.Write([]string{
			l.CreatedAt.Format(time.RFC3339),
			dn,
			l.Action,
			l.ResourceType,
			resourceID,
			groupID,
			l.IPAddress,
			lat,
			lng,
		}); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flush csv: %w", err)
	}
	return buf.Bytes(), nil
}

// ExportJSON serializes audit log entries as a JSON array.
func (s *AuditService) ExportJSON(logs []model.AuditLogWithUser) ([]byte, error) {
	resp := make([]model.AuditLogResponse, len(logs))
	for i := range logs {
		resp[i] = logs[i].ToResponse()
	}
	return json.Marshal(resp)
}
