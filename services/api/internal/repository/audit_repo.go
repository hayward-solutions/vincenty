package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vincenty/api/internal/model"
)

// AuditRepository handles database operations for audit logs.
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository creates a new AuditRepository.
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// Create inserts a new audit log entry.
func (r *AuditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, user_id, device_id, action, resource_type, resource_id, group_id, metadata, location, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8,
			CASE WHEN $9::double precision IS NOT NULL AND $10::double precision IS NOT NULL
				THEN ST_SetSRID(ST_MakePoint($10, $9), 4326)
				ELSE NULL
			END,
			$11, $12)`

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	_, err := r.pool.Exec(ctx, query,
		log.ID, log.UserID, log.DeviceID,
		log.Action, log.ResourceType, log.ResourceID, log.GroupID,
		log.Metadata, log.Lat, log.Lng,
		log.IPAddress, log.CreatedAt,
	)
	return err
}

// selectBase is the common SELECT for all list queries.
const auditSelectBase = `
	SELECT al.id, al.user_id, al.device_id,
		al.action, al.resource_type, al.resource_id, al.group_id,
		al.metadata,
		ST_Y(al.location) AS lat, ST_X(al.location) AS lng,
		host(al.ip_address), al.created_at,
		u.username, u.display_name
	FROM audit_logs al
	INNER JOIN users u ON u.id = al.user_id`

const auditCountBase = `
	SELECT COUNT(*)
	FROM audit_logs al
	INNER JOIN users u ON u.id = al.user_id`

// ListByUser returns audit logs for a single user with pagination.
func (r *AuditRepository) ListByUser(ctx context.Context, userID uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
	where := " WHERE al.user_id = $1"
	args := []any{userID}
	where, args = appendFilters(where, args, f)
	return r.list(ctx, where, args, f)
}

// ListByGroup returns audit logs scoped to a group with pagination.
func (r *AuditRepository) ListByGroup(ctx context.Context, groupID uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
	where := " WHERE al.group_id = $1"
	args := []any{groupID}
	where, args = appendFilters(where, args, f)
	return r.list(ctx, where, args, f)
}

// ListAll returns all audit logs with pagination (admin).
func (r *AuditRepository) ListAll(ctx context.Context, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
	where := " WHERE 1=1"
	var args []any
	where, args = appendFilters(where, args, f)
	return r.list(ctx, where, args, f)
}

// list executes the query with the given WHERE clause and returns results + total count.
func (r *AuditRepository) list(ctx context.Context, where string, args []any, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
	// Count
	var total int
	countQuery := auditCountBase + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	// Data
	offset := (f.Page - 1) * f.PageSize
	args = append(args, f.PageSize, offset)
	dataQuery := auditSelectBase + where +
		" ORDER BY al.created_at DESC" +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []model.AuditLogWithUser
	for rows.Next() {
		var l model.AuditLogWithUser
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.DeviceID,
			&l.Action, &l.ResourceType, &l.ResourceID, &l.GroupID,
			&l.Metadata,
			&l.Lat, &l.Lng,
			&l.IPAddress, &l.CreatedAt,
			&l.Username, &l.DisplayName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

// appendFilters adds optional WHERE clauses for time range, action, and resource type.
func appendFilters(where string, args []any, f model.AuditFilters) (string, []any) {
	var clauses []string
	if f.From != nil {
		args = append(args, *f.From)
		clauses = append(clauses, fmt.Sprintf("al.created_at >= $%d", len(args)))
	}
	if f.To != nil {
		args = append(args, *f.To)
		clauses = append(clauses, fmt.Sprintf("al.created_at <= $%d", len(args)))
	}
	if f.Action != "" {
		args = append(args, f.Action)
		clauses = append(clauses, fmt.Sprintf("al.action = $%d", len(args)))
	}
	if f.ResourceType != "" {
		args = append(args, f.ResourceType)
		clauses = append(clauses, fmt.Sprintf("al.resource_type = $%d", len(args)))
	}
	if len(clauses) > 0 {
		where += " AND " + strings.Join(clauses, " AND ")
	}
	return where, args
}
