package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vincenty/api/internal/model"
)

// MessageRepository handles database operations for messages and attachments.
type MessageRepository struct {
	pool *pgxpool.Pool
}

// NewMessageRepository creates a new MessageRepository.
func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

// Create inserts a new message. If lat/lng are provided, a PostGIS point is stored.
func (r *MessageRepository) Create(ctx context.Context, msg *model.Message) error {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}

	var query string
	var args []any

	if msg.Lat != nil && msg.Lng != nil {
		query = `
			INSERT INTO messages (id, sender_id, sender_device_id, group_id, recipient_id,
			                      content, message_type, location, metadata, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7,
			        ST_SetSRID(ST_MakePoint($8, $9), 4326), $10, NOW())
			RETURNING created_at`
		args = []any{
			msg.ID, msg.SenderID, msg.SenderDeviceID, msg.GroupID, msg.RecipientID,
			msg.Content, msg.MessageType, *msg.Lng, *msg.Lat, msg.Metadata,
		}
	} else {
		query = `
			INSERT INTO messages (id, sender_id, sender_device_id, group_id, recipient_id,
			                      content, message_type, metadata, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
			RETURNING created_at`
		args = []any{
			msg.ID, msg.SenderID, msg.SenderDeviceID, msg.GroupID, msg.RecipientID,
			msg.Content, msg.MessageType, msg.Metadata,
		}
	}

	return r.pool.QueryRow(ctx, query, args...).Scan(&msg.CreatedAt)
}

// CreateAttachment inserts an attachment record linked to a message.
func (r *MessageRepository) CreateAttachment(ctx context.Context, att *model.Attachment) error {
	if att.ID == uuid.Nil {
		att.ID = uuid.New()
	}

	query := `
		INSERT INTO attachments (id, message_id, filename, content_type, size_bytes, object_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`

	return r.pool.QueryRow(ctx, query,
		att.ID, att.MessageID, att.Filename, att.ContentType, att.SizeBytes, att.ObjectKey,
	).Scan(&att.CreatedAt)
}

// GetByID retrieves a message by ID, joined with sender details and attachments.
func (r *MessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MessageWithUser, error) {
	query := `
		SELECT m.id, m.sender_id, m.sender_device_id, m.group_id, m.recipient_id,
		       m.content, m.message_type,
		       ST_Y(m.location) AS lat, ST_X(m.location) AS lng,
		       m.metadata, m.created_at,
		       u.username, u.display_name
		FROM messages m
		INNER JOIN users u ON u.id = m.sender_id
		WHERE m.id = $1`

	mwu := &model.MessageWithUser{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&mwu.ID, &mwu.SenderID, &mwu.SenderDeviceID, &mwu.GroupID, &mwu.RecipientID,
		&mwu.Content, &mwu.MessageType,
		&mwu.Lat, &mwu.Lng,
		&mwu.Metadata, &mwu.CreatedAt,
		&mwu.Username, &mwu.DisplayName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("message")
		}
		return nil, err
	}

	// Load attachments
	atts, err := r.getAttachmentsByMessageID(ctx, id)
	if err != nil {
		return nil, err
	}
	mwu.Attachments = atts

	return mwu, nil
}

// ListByGroup returns messages for a group, cursor-based (before a given message time), newest first.
// Results include sender info and attachments.
func (r *MessageRepository) ListByGroup(ctx context.Context, groupID uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error) {
	var query string
	var args []any

	if before != nil {
		query = `
			SELECT m.id, m.sender_id, m.sender_device_id, m.group_id, m.recipient_id,
			       m.content, m.message_type,
			       ST_Y(m.location) AS lat, ST_X(m.location) AS lng,
			       m.metadata, m.created_at,
			       u.username, u.display_name
			FROM messages m
			INNER JOIN users u ON u.id = m.sender_id
			WHERE m.group_id = $1
			  AND m.created_at < (SELECT created_at FROM messages WHERE id = $2)
			ORDER BY m.created_at DESC
			LIMIT $3`
		args = []any{groupID, *before, limit}
	} else {
		query = `
			SELECT m.id, m.sender_id, m.sender_device_id, m.group_id, m.recipient_id,
			       m.content, m.message_type,
			       ST_Y(m.location) AS lat, ST_X(m.location) AS lng,
			       m.metadata, m.created_at,
			       u.username, u.display_name
			FROM messages m
			INNER JOIN users u ON u.id = m.sender_id
			WHERE m.group_id = $1
			ORDER BY m.created_at DESC
			LIMIT $2`
		args = []any{groupID, limit}
	}

	return r.scanMessagesWithAttachments(ctx, query, args...)
}

// ListDirect returns direct messages between two users, cursor-based, newest first.
func (r *MessageRepository) ListDirect(ctx context.Context, userA, userB uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error) {
	var query string
	var args []any

	if before != nil {
		query = `
			SELECT m.id, m.sender_id, m.sender_device_id, m.group_id, m.recipient_id,
			       m.content, m.message_type,
			       ST_Y(m.location) AS lat, ST_X(m.location) AS lng,
			       m.metadata, m.created_at,
			       u.username, u.display_name
			FROM messages m
			INNER JOIN users u ON u.id = m.sender_id
			WHERE m.group_id IS NULL
			  AND ((m.sender_id = $1 AND m.recipient_id = $2)
			       OR (m.sender_id = $2 AND m.recipient_id = $1))
			  AND m.created_at < (SELECT created_at FROM messages WHERE id = $3)
			ORDER BY m.created_at DESC
			LIMIT $4`
		args = []any{userA, userB, *before, limit}
	} else {
		query = `
			SELECT m.id, m.sender_id, m.sender_device_id, m.group_id, m.recipient_id,
			       m.content, m.message_type,
			       ST_Y(m.location) AS lat, ST_X(m.location) AS lng,
			       m.metadata, m.created_at,
			       u.username, u.display_name
			FROM messages m
			INNER JOIN users u ON u.id = m.sender_id
			WHERE m.group_id IS NULL
			  AND ((m.sender_id = $1 AND m.recipient_id = $2)
			       OR (m.sender_id = $2 AND m.recipient_id = $1))
			ORDER BY m.created_at DESC
			LIMIT $3`
		args = []any{userA, userB, limit}
	}

	return r.scanMessagesWithAttachments(ctx, query, args...)
}

// Delete removes a message by ID. Returns NotFoundError if the message does not exist.
func (r *MessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM messages WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("message")
	}
	return nil
}

// GetAttachmentByID retrieves a single attachment by its ID.
func (r *MessageRepository) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*model.Attachment, error) {
	query := `
		SELECT id, message_id, filename, content_type, size_bytes, object_key, created_at
		FROM attachments WHERE id = $1`

	att := &model.Attachment{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&att.ID, &att.MessageID, &att.Filename, &att.ContentType,
		&att.SizeBytes, &att.ObjectKey, &att.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("attachment")
		}
		return nil, err
	}
	return att, nil
}

// GetAttachmentObjectKeys returns the object keys for all attachments of a message.
// Used before deleting a message to clean up S3 objects.
func (r *MessageRepository) GetAttachmentObjectKeys(ctx context.Context, messageID uuid.UUID) ([]string, error) {
	query := `SELECT object_key FROM attachments WHERE message_id = $1`
	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// DMPartner represents a user the caller has exchanged direct messages with.
type DMPartner struct {
	UserID      uuid.UUID
	Username    string
	DisplayName *string
}

// ListDMPartners returns distinct users the caller has DM history with.
func (r *MessageRepository) ListDMPartners(ctx context.Context, userID uuid.UUID) ([]DMPartner, error) {
	query := `
		SELECT DISTINCT partner_id, u.username, u.display_name
		FROM (
			SELECT recipient_id AS partner_id FROM messages
			WHERE sender_id = $1 AND group_id IS NULL
			UNION
			SELECT sender_id AS partner_id FROM messages
			WHERE recipient_id = $1 AND group_id IS NULL
		) dm
		INNER JOIN users u ON u.id = dm.partner_id
		ORDER BY u.username ASC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var partners []DMPartner
	for rows.Next() {
		var p DMPartner
		if err := rows.Scan(&p.UserID, &p.Username, &p.DisplayName); err != nil {
			return nil, err
		}
		partners = append(partners, p)
	}
	return partners, rows.Err()
}

// --------------------------------------------------------------------------
// Internal helpers
// --------------------------------------------------------------------------

// getAttachmentsByMessageID loads all attachments for a single message.
func (r *MessageRepository) getAttachmentsByMessageID(ctx context.Context, messageID uuid.UUID) ([]model.Attachment, error) {
	query := `
		SELECT id, message_id, filename, content_type, size_bytes, object_key, created_at
		FROM attachments
		WHERE message_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var atts []model.Attachment
	for rows.Next() {
		var a model.Attachment
		if err := rows.Scan(
			&a.ID, &a.MessageID, &a.Filename, &a.ContentType,
			&a.SizeBytes, &a.ObjectKey, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		atts = append(atts, a)
	}
	return atts, rows.Err()
}

// scanMessagesWithAttachments runs a message query and batch-loads attachments.
func (r *MessageRepository) scanMessagesWithAttachments(ctx context.Context, query string, args ...any) ([]model.MessageWithUser, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []model.MessageWithUser
	var messageIDs []uuid.UUID

	for rows.Next() {
		var m model.MessageWithUser
		if err := rows.Scan(
			&m.ID, &m.SenderID, &m.SenderDeviceID, &m.GroupID, &m.RecipientID,
			&m.Content, &m.MessageType,
			&m.Lat, &m.Lng,
			&m.Metadata, &m.CreatedAt,
			&m.Username, &m.DisplayName,
		); err != nil {
			return nil, err
		}
		messages = append(messages, m)
		messageIDs = append(messageIDs, m.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return messages, nil
	}

	// Batch-load attachments for all messages
	attQuery := `
		SELECT id, message_id, filename, content_type, size_bytes, object_key, created_at
		FROM attachments
		WHERE message_id = ANY($1)
		ORDER BY created_at ASC`

	attRows, err := r.pool.Query(ctx, attQuery, messageIDs)
	if err != nil {
		return nil, err
	}
	defer attRows.Close()

	attMap := make(map[uuid.UUID][]model.Attachment)
	for attRows.Next() {
		var a model.Attachment
		if err := attRows.Scan(
			&a.ID, &a.MessageID, &a.Filename, &a.ContentType,
			&a.SizeBytes, &a.ObjectKey, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		attMap[a.MessageID] = append(attMap[a.MessageID], a)
	}
	if err := attRows.Err(); err != nil {
		return nil, err
	}

	// Assign attachments to messages
	for i := range messages {
		messages[i].Attachments = attMap[messages[i].ID]
	}

	return messages, nil
}
