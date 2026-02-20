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
func (r *DeviceRepository) Create(ctx context.Context, device *model.Device) error {
	query := `
		INSERT INTO devices (id, user_id, name, device_type, device_uid)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`

	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	if device.DeviceUID == nil {
		uid := uuid.New().String()
		device.DeviceUID = &uid
	}

	return r.pool.QueryRow(ctx, query,
		device.ID, device.UserID, device.Name, device.DeviceType, device.DeviceUID,
	).Scan(&device.CreatedAt, &device.UpdatedAt)
}

// GetByID retrieves a device by ID.
func (r *DeviceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	query := `
		SELECT id, user_id, name, device_type, device_uid, last_seen_at, created_at, updated_at
		FROM devices WHERE id = $1`

	d := &model.Device{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.UserID, &d.Name, &d.DeviceType, &d.DeviceUID,
		&d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt,
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
		SELECT id, user_id, name, device_type, device_uid, last_seen_at, created_at, updated_at
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
			&d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt,
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
		UPDATE devices SET name = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query, device.ID, device.Name).Scan(&device.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound("device")
		}
		return err
	}
	return nil
}

// GetByDeviceUID retrieves a device by its unique device UID (e.g., CoT event UID).
func (r *DeviceRepository) GetByDeviceUID(ctx context.Context, deviceUID string) (*model.Device, error) {
	query := `
		SELECT id, user_id, name, device_type, device_uid, last_seen_at, created_at, updated_at
		FROM devices WHERE device_uid = $1`

	d := &model.Device{}
	err := r.pool.QueryRow(ctx, query, deviceUID).Scan(
		&d.ID, &d.UserID, &d.Name, &d.DeviceType, &d.DeviceUID,
		&d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("device")
		}
		return nil, err
	}
	return d, nil
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
