package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// MediaRoomRepository handles database operations for media rooms.
type MediaRoomRepository struct {
	pool *pgxpool.Pool
}

// NewMediaRoomRepository creates a new MediaRoomRepository.
func NewMediaRoomRepository(pool *pgxpool.Pool) *MediaRoomRepository {
	return &MediaRoomRepository{pool: pool}
}

// Create inserts a new media room.
func (r *MediaRoomRepository) Create(ctx context.Context, room *model.MediaRoom) error {
	query := `
		INSERT INTO media_rooms (id, name, room_type, group_id, created_by, livekit_room, is_active, max_participants, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at`

	if room.ID == uuid.Nil {
		room.ID = uuid.New()
	}

	return r.pool.QueryRow(ctx, query,
		room.ID, room.Name, room.RoomType, room.GroupID,
		room.CreatedBy, room.LiveKitRoom, room.IsActive,
		room.MaxParticipants, room.Metadata,
	).Scan(&room.CreatedAt)
}

// GetByID retrieves a media room by its ID.
func (r *MediaRoomRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MediaRoom, error) {
	query := `
		SELECT id, name, room_type, group_id, created_by, livekit_room,
		       is_active, max_participants, metadata, created_at, ended_at
		FROM media_rooms WHERE id = $1`

	room := &model.MediaRoom{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&room.ID, &room.Name, &room.RoomType, &room.GroupID,
		&room.CreatedBy, &room.LiveKitRoom, &room.IsActive,
		&room.MaxParticipants, &room.Metadata, &room.CreatedAt, &room.EndedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("media room")
		}
		return nil, err
	}
	return room, nil
}

// GetByLiveKitRoom retrieves a media room by its LiveKit room name.
func (r *MediaRoomRepository) GetByLiveKitRoom(ctx context.Context, lkRoom string) (*model.MediaRoom, error) {
	query := `
		SELECT id, name, room_type, group_id, created_by, livekit_room,
		       is_active, max_participants, metadata, created_at, ended_at
		FROM media_rooms WHERE livekit_room = $1`

	room := &model.MediaRoom{}
	err := r.pool.QueryRow(ctx, query, lkRoom).Scan(
		&room.ID, &room.Name, &room.RoomType, &room.GroupID,
		&room.CreatedBy, &room.LiveKitRoom, &room.IsActive,
		&room.MaxParticipants, &room.Metadata, &room.CreatedAt, &room.EndedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("media room")
		}
		return nil, err
	}
	return room, nil
}

// ListByGroupID retrieves media rooms for a group.
func (r *MediaRoomRepository) ListByGroupID(ctx context.Context, groupID uuid.UUID, activeOnly bool) ([]model.MediaRoom, error) {
	query := `
		SELECT id, name, room_type, group_id, created_by, livekit_room,
		       is_active, max_participants, metadata, created_at, ended_at
		FROM media_rooms
		WHERE group_id = $1`
	if activeOnly {
		query += ` AND is_active = true`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []model.MediaRoom
	for rows.Next() {
		var room model.MediaRoom
		if err := rows.Scan(
			&room.ID, &room.Name, &room.RoomType, &room.GroupID,
			&room.CreatedBy, &room.LiveKitRoom, &room.IsActive,
			&room.MaxParticipants, &room.Metadata, &room.CreatedAt, &room.EndedAt,
		); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

// ListActiveByUserGroups retrieves all active media rooms in groups the user belongs to.
func (r *MediaRoomRepository) ListActiveByUserGroups(ctx context.Context, userID uuid.UUID) ([]model.MediaRoom, error) {
	query := `
		SELECT mr.id, mr.name, mr.room_type, mr.group_id, mr.created_by, mr.livekit_room,
		       mr.is_active, mr.max_participants, mr.metadata, mr.created_at, mr.ended_at
		FROM media_rooms mr
		INNER JOIN group_members gm ON gm.group_id = mr.group_id
		WHERE gm.user_id = $1 AND mr.is_active = true
		ORDER BY mr.created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []model.MediaRoom
	for rows.Next() {
		var room model.MediaRoom
		if err := rows.Scan(
			&room.ID, &room.Name, &room.RoomType, &room.GroupID,
			&room.CreatedBy, &room.LiveKitRoom, &room.IsActive,
			&room.MaxParticipants, &room.Metadata, &room.CreatedAt, &room.EndedAt,
		); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

// Update modifies an existing media room.
func (r *MediaRoomRepository) Update(ctx context.Context, room *model.MediaRoom) error {
	query := `
		UPDATE media_rooms
		SET name = $2, is_active = $3, max_participants = $4, metadata = $5, ended_at = $6
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		room.ID, room.Name, room.IsActive,
		room.MaxParticipants, room.Metadata, room.EndedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("media room")
	}
	return nil
}

// End marks a media room as ended.
func (r *MediaRoomRepository) End(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE media_rooms
		SET is_active = false, ended_at = NOW()
		WHERE id = $1 AND is_active = true`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("media room")
	}
	return nil
}

// --------------------------------------------------------------------------
// Participants
// --------------------------------------------------------------------------

// AddParticipant records a user joining a media room.
func (r *MediaRoomRepository) AddParticipant(ctx context.Context, p *model.MediaRoomParticipant) error {
	query := `
		INSERT INTO media_room_participants (id, room_id, user_id, device_id, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING joined_at`

	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	return r.pool.QueryRow(ctx, query,
		p.ID, p.RoomID, p.UserID, p.DeviceID, p.Role,
	).Scan(&p.JoinedAt)
}

// MarkParticipantLeft records a user leaving a media room.
func (r *MediaRoomRepository) MarkParticipantLeft(ctx context.Context, roomID, userID uuid.UUID) error {
	query := `
		UPDATE media_room_participants
		SET left_at = NOW()
		WHERE room_id = $1 AND user_id = $2 AND left_at IS NULL`

	_, err := r.pool.Exec(ctx, query, roomID, userID)
	return err
}

// ListParticipants retrieves all participants (current and past) in a room.
func (r *MediaRoomRepository) ListParticipants(ctx context.Context, roomID uuid.UUID) ([]model.MediaRoomParticipant, error) {
	query := `
		SELECT id, room_id, user_id, device_id, role, joined_at, left_at
		FROM media_room_participants
		WHERE room_id = $1
		ORDER BY joined_at ASC`

	rows, err := r.pool.Query(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []model.MediaRoomParticipant
	for rows.Next() {
		var p model.MediaRoomParticipant
		if err := rows.Scan(
			&p.ID, &p.RoomID, &p.UserID, &p.DeviceID,
			&p.Role, &p.JoinedAt, &p.LeftAt,
		); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

// CountActiveParticipants returns the number of currently active participants.
func (r *MediaRoomRepository) CountActiveParticipants(ctx context.Context, roomID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM media_room_participants WHERE room_id = $1 AND left_at IS NULL`
	err := r.pool.QueryRow(ctx, query, roomID).Scan(&count)
	return count, err
}
