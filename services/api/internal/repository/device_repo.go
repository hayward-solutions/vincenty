package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// DeviceRepository handles database operations for devices.
type DeviceRepository struct {
	pool *pgxpool.Pool
}

// NewDeviceRepository creates a new DeviceRepository.
func NewDeviceRepository(pool *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{pool: pool}
}

// Create inserts a new device into the database.
// If this is the user's first device, it is automatically set as primary.
func (r *DeviceRepository) Create(ctx context.Context, device *model.Device) error {
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	if device.DeviceUID == nil {
		uid := uuid.New().String()
		device.DeviceUID = &uid
	}

	// Check if the user already has any devices — if not, make this one primary.
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM devices WHERE user_id = $1`, device.UserID).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		device.IsPrimary = true
	}

	query := `
		INSERT INTO devices (id, user_id, name, device_type, device_uid, user_agent, app_version, is_primary)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		device.ID, device.UserID, device.Name, device.DeviceType, device.DeviceUID, device.UserAgent, device.AppVersion, device.IsPrimary,
	).Scan(&device.CreatedAt, &device.UpdatedAt)
}

// GetByID retrieves a device by ID.
func (r *DeviceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	query := `
		SELECT id, user_id, name, device_type, device_uid, user_agent, app_version, is_primary, last_seen_at, created_at, updated_at
		FROM devices WHERE id = $1`

	d := &model.Device{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.UserID, &d.Name, &d.DeviceType, &d.DeviceUID,
		&d.UserAgent, &d.AppVersion, &d.IsPrimary, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("device")
		}
		return nil, err
	}
	return d, nil
}

// ListByUserID retrieves all devices for a user.
func (r *DeviceRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Device, error) {
	query := `
		SELECT id, user_id, name, device_type, device_uid, user_agent, app_version, is_primary, last_seen_at, created_at, updated_at
		FROM devices WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		var d model.Device
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Name, &d.DeviceType, &d.DeviceUID,
			&d.UserAgent, &d.AppVersion, &d.IsPrimary, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}

	return devices, rows.Err()
}

// Update modifies an existing device.
func (r *DeviceRepository) Update(ctx context.Context, device *model.Device) error {
	query := `
		UPDATE devices SET name = $2, user_agent = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query, device.ID, device.Name, device.UserAgent).Scan(&device.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound("device")
		}
		return err
	}
	return nil
}

// TouchLastSeen updates last_seen_at, optionally user_agent, and optionally
// app_version for a device. Nil values are preserved (COALESCE).
func (r *DeviceRepository) TouchLastSeen(ctx context.Context, id uuid.UUID, userAgent *string, appVersion *string) error {
	query := `
		UPDATE devices
		SET last_seen_at = NOW(),
		    user_agent   = COALESCE($2, user_agent),
		    app_version  = COALESCE($3, app_version),
		    updated_at   = NOW()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id, userAgent, appVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("device")
	}
	return nil
}

// GetByDeviceUID retrieves a device by its unique device UID (e.g., CoT event UID).
func (r *DeviceRepository) GetByDeviceUID(ctx context.Context, deviceUID string) (*model.Device, error) {
	query := `
		SELECT id, user_id, name, device_type, device_uid, user_agent, app_version, is_primary, last_seen_at, created_at, updated_at
		FROM devices WHERE device_uid = $1`

	d := &model.Device{}
	err := r.pool.QueryRow(ctx, query, deviceUID).Scan(
		&d.ID, &d.UserID, &d.Name, &d.DeviceType, &d.DeviceUID,
		&d.UserAgent, &d.AppVersion, &d.IsPrimary, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("device")
		}
		return nil, err
	}
	return d, nil
}

// FindSingleByUserAgent looks for an existing device matching the user, device
// type, and user-agent string. It returns the device only when exactly one
// candidate exists — if there are zero or multiple matches it returns nil so
// the caller can fall through to creating a new device.
func (r *DeviceRepository) FindSingleByUserAgent(ctx context.Context, userID uuid.UUID, deviceType, userAgent string) (*model.Device, error) {
	query := `
		SELECT id, user_id, name, device_type, device_uid, user_agent, app_version, is_primary, last_seen_at, created_at, updated_at
		FROM devices
		WHERE user_id = $1 AND device_type = $2 AND user_agent = $3`

	rows, err := r.pool.Query(ctx, query, userID, deviceType, userAgent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result *model.Device
	count := 0
	for rows.Next() {
		count++
		if count > 1 {
			// Multiple matches — ambiguous, bail out.
			return nil, nil
		}
		d := &model.Device{}
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Name, &d.DeviceType, &d.DeviceUID,
			&d.UserAgent, &d.AppVersion, &d.IsPrimary, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = d
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// SetPrimary sets the given device as the user's primary device, unsetting any
// previously primary device for that user. Both operations run in a transaction.
func (r *DeviceRepository) SetPrimary(ctx context.Context, userID, deviceID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Unset current primary
	if _, err := tx.Exec(ctx, `UPDATE devices SET is_primary = false, updated_at = NOW() WHERE user_id = $1 AND is_primary = true`, userID); err != nil {
		return err
	}

	// Set new primary
	tag, err := tx.Exec(ctx, `UPDATE devices SET is_primary = true, updated_at = NOW() WHERE id = $1 AND user_id = $2`, deviceID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("device")
	}

	return tx.Commit(ctx)
}

// Delete removes a device by ID.
func (r *DeviceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM devices WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("device")
	}
	return nil
}
