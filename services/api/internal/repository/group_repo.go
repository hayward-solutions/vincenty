package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// GroupRepository handles database operations for groups.
type GroupRepository struct {
	pool *pgxpool.Pool
}

// NewGroupRepository creates a new GroupRepository.
func NewGroupRepository(pool *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{pool: pool}
}

// Create inserts a new group into the database.
func (r *GroupRepository) Create(ctx context.Context, group *model.Group) error {
	query := `
		INSERT INTO groups (id, name, description, marker_icon, marker_color, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`

	if group.ID == uuid.Nil {
		group.ID = uuid.New()
	}
	if group.MarkerIcon == "" {
		group.MarkerIcon = "circle"
	}
	if group.MarkerColor == "" {
		group.MarkerColor = "#3b82f6"
	}

	return r.pool.QueryRow(ctx, query,
		group.ID, group.Name, group.Description,
		group.MarkerIcon, group.MarkerColor, group.CreatedBy,
	).Scan(&group.CreatedAt, &group.UpdatedAt)
}

// GetByID retrieves a group by its ID.
func (r *GroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Group, error) {
	query := `
		SELECT id, name, description, marker_icon, marker_color, created_by, created_at, updated_at
		FROM groups WHERE id = $1`

	g := &model.Group{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&g.ID, &g.Name, &g.Description,
		&g.MarkerIcon, &g.MarkerColor,
		&g.CreatedBy, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("group")
		}
		return nil, err
	}
	return g, nil
}

// List retrieves a paginated list of all groups with member counts.
func (r *GroupRepository) List(ctx context.Context, page, pageSize int) ([]model.Group, []int, int, error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM groups`
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, nil, 0, err
	}

	// Fetch page with member counts
	offset := (page - 1) * pageSize
	query := `
		SELECT g.id, g.name, g.description, g.marker_icon, g.marker_color,
		       g.created_by, g.created_at, g.updated_at,
		       COALESCE(mc.cnt, 0) AS member_count
		FROM groups g
		LEFT JOIN (
			SELECT group_id, COUNT(*) AS cnt FROM group_members GROUP BY group_id
		) mc ON mc.group_id = g.id
		ORDER BY g.created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, nil, 0, err
	}
	defer rows.Close()

	var groups []model.Group
	var counts []int
	for rows.Next() {
		var g model.Group
		var count int
		if err := rows.Scan(
			&g.ID, &g.Name, &g.Description,
			&g.MarkerIcon, &g.MarkerColor,
			&g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &count,
		); err != nil {
			return nil, nil, 0, err
		}
		groups = append(groups, g)
		counts = append(counts, count)
	}

	return groups, counts, total, rows.Err()
}

// ListByUserID retrieves groups that a user is a member of.
func (r *GroupRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Group, []int, error) {
	query := `
		SELECT g.id, g.name, g.description, g.marker_icon, g.marker_color,
		       g.created_by, g.created_at, g.updated_at,
		       COALESCE(mc.cnt, 0) AS member_count
		FROM groups g
		INNER JOIN group_members gm ON gm.group_id = g.id
		LEFT JOIN (
			SELECT group_id, COUNT(*) AS cnt FROM group_members GROUP BY group_id
		) mc ON mc.group_id = g.id
		WHERE gm.user_id = $1
		ORDER BY g.name ASC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var groups []model.Group
	var counts []int
	for rows.Next() {
		var g model.Group
		var count int
		if err := rows.Scan(
			&g.ID, &g.Name, &g.Description,
			&g.MarkerIcon, &g.MarkerColor,
			&g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &count,
		); err != nil {
			return nil, nil, err
		}
		groups = append(groups, g)
		counts = append(counts, count)
	}

	return groups, counts, rows.Err()
}

// Update modifies an existing group.
func (r *GroupRepository) Update(ctx context.Context, group *model.Group) error {
	query := `
		UPDATE groups
		SET name = $2, description = $3, marker_icon = $4, marker_color = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		group.ID, group.Name, group.Description,
		group.MarkerIcon, group.MarkerColor,
	).Scan(&group.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound("group")
		}
		return err
	}
	return nil
}

// UpdateMarker updates only the marker icon and color fields for a group.
func (r *GroupRepository) UpdateMarker(ctx context.Context, id uuid.UUID, markerIcon, markerColor string) (*model.Group, error) {
	query := `
		UPDATE groups
		SET marker_icon = $2, marker_color = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, description, marker_icon, marker_color, created_by, created_at, updated_at`

	g := &model.Group{}
	err := r.pool.QueryRow(ctx, query, id, markerIcon, markerColor).Scan(
		&g.ID, &g.Name, &g.Description,
		&g.MarkerIcon, &g.MarkerColor,
		&g.CreatedBy, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("group")
		}
		return nil, err
	}
	return g, nil
}

// Delete removes a group by ID.
func (r *GroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM groups WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("group")
	}
	return nil
}

// MemberCount returns the number of members in a group.
func (r *GroupRepository) MemberCount(ctx context.Context, groupID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM group_members WHERE group_id = $1`
	err := r.pool.QueryRow(ctx, query, groupID).Scan(&count)
	return count, err
}

// --------------------------------------------------------------------------
// Group Members
// --------------------------------------------------------------------------

// AddMember adds a user to a group.
func (r *GroupRepository) AddMember(ctx context.Context, member *model.GroupMember) error {
	query := `
		INSERT INTO group_members (id, group_id, user_id, can_read, can_write, is_group_admin)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`

	if member.ID == uuid.Nil {
		member.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		member.ID, member.GroupID, member.UserID,
		member.CanRead, member.CanWrite, member.IsGroupAdmin,
	).Scan(&member.CreatedAt, &member.UpdatedAt)
	if err != nil {
		// Check for unique constraint violation (user already in group)
		if isDuplicateKeyError(err) {
			return model.ErrConflict("user is already a member of this group")
		}
		return err
	}
	return nil
}

// GetMember retrieves a specific group membership.
func (r *GroupRepository) GetMember(ctx context.Context, groupID, userID uuid.UUID) (*model.GroupMember, error) {
	query := `
		SELECT id, group_id, user_id, can_read, can_write, is_group_admin, created_at, updated_at
		FROM group_members
		WHERE group_id = $1 AND user_id = $2`

	m := &model.GroupMember{}
	err := r.pool.QueryRow(ctx, query, groupID, userID).Scan(
		&m.ID, &m.GroupID, &m.UserID,
		&m.CanRead, &m.CanWrite, &m.IsGroupAdmin,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("group member")
		}
		return nil, err
	}
	return m, nil
}

// GetMemberByID retrieves a group membership by its own ID.
func (r *GroupRepository) GetMemberByID(ctx context.Context, memberID uuid.UUID) (*model.GroupMember, error) {
	query := `
		SELECT id, group_id, user_id, can_read, can_write, is_group_admin, created_at, updated_at
		FROM group_members
		WHERE id = $1`

	m := &model.GroupMember{}
	err := r.pool.QueryRow(ctx, query, memberID).Scan(
		&m.ID, &m.GroupID, &m.UserID,
		&m.CanRead, &m.CanWrite, &m.IsGroupAdmin,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("group member")
		}
		return nil, err
	}
	return m, nil
}

// ListMembers retrieves all members of a group with user details.
func (r *GroupRepository) ListMembers(ctx context.Context, groupID uuid.UUID) ([]model.GroupMemberWithUser, error) {
	query := `
		SELECT gm.id, gm.group_id, gm.user_id,
		       gm.can_read, gm.can_write, gm.is_group_admin,
		       gm.created_at, gm.updated_at,
		       u.username, u.display_name
		FROM group_members gm
		INNER JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1
		ORDER BY u.username ASC`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.GroupMemberWithUser
	for rows.Next() {
		var m model.GroupMemberWithUser
		if err := rows.Scan(
			&m.ID, &m.GroupID, &m.UserID,
			&m.CanRead, &m.CanWrite, &m.IsGroupAdmin,
			&m.CreatedAt, &m.UpdatedAt,
			&m.Username, &m.DisplayName,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, rows.Err()
}

// UpdateMember modifies a group membership.
func (r *GroupRepository) UpdateMember(ctx context.Context, member *model.GroupMember) error {
	query := `
		UPDATE group_members
		SET can_read = $2, can_write = $3, is_group_admin = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		member.ID, member.CanRead, member.CanWrite, member.IsGroupAdmin,
	).Scan(&member.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound("group member")
		}
		return err
	}
	return nil
}

// RemoveMember removes a user from a group.
func (r *GroupRepository) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	query := `DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, query, groupID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("group member")
	}
	return nil
}

// isDuplicateKeyError checks if a pgx error is a unique constraint violation.
func isDuplicateKeyError(err error) bool {
	// pgx wraps errors; check the error message for the Postgres unique_violation code
	return err != nil && (errors.Is(err, pgx.ErrNoRows) == false) &&
		containsDuplicateKey(err.Error())
}

func containsDuplicateKey(msg string) bool {
	// Postgres error code 23505 = unique_violation
	return len(msg) > 0 && (contains(msg, "23505") || contains(msg, "duplicate key"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
