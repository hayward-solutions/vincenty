package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LocationRecord represents a single row from the location_history table.
type LocationRecord struct {
	UserID      uuid.UUID
	DeviceID    uuid.UUID
	Lat         float64
	Lng         float64
	Altitude    *float64
	Heading     *float64
	Speed       *float64
	Accuracy    *float64
	RecordedAt  time.Time
	Username    string  // populated by join queries
	DisplayName *string // populated by join queries
	DeviceName  *string // populated by join queries (from devices table)
	IsPrimary   bool    // populated by join queries (from devices table)
}

// LocationRepository handles database operations for location data.
type LocationRepository struct {
	pool *pgxpool.Pool
}

// NewLocationRepository creates a new LocationRepository.
func NewLocationRepository(pool *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{pool: pool}
}

// Create inserts a new location record into location_history.
func (r *LocationRepository) Create(ctx context.Context, userID, deviceID uuid.UUID, lat, lng float64, altitude, heading, speed, accuracy *float64) error {
	query := `
		INSERT INTO location_history (id, user_id, device_id, location, altitude, heading, speed, accuracy, recorded_at)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326), $6, $7, $8, $9, NOW())`

	_, err := r.pool.Exec(ctx, query,
		uuid.New(), userID, deviceID, lng, lat, altitude, heading, speed, accuracy,
	)
	return err
}

// UpdateDeviceLocation updates the device's last known location and last seen time.
func (r *LocationRepository) UpdateDeviceLocation(ctx context.Context, deviceID uuid.UUID, lat, lng float64) error {
	query := `
		UPDATE devices
		SET last_location = ST_SetSRID(ST_MakePoint($2, $3), 4326),
		    last_seen_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, deviceID, lng, lat)
	return err
}

// GetLatestByGroup returns the most recent location for each device in a group.
// This is used to send a snapshot when a client connects.
func (r *LocationRepository) GetLatestByGroup(ctx context.Context, groupID uuid.UUID) ([]LocationRecord, error) {
	query := `
		SELECT DISTINCT ON (lh.device_id)
			lh.user_id, lh.device_id,
			ST_Y(lh.location) AS lat, ST_X(lh.location) AS lng,
			lh.altitude, lh.heading, lh.speed, lh.accuracy,
			lh.recorded_at,
			u.username, u.display_name,
			d.name AS device_name,
			COALESCE(d.is_primary, false) AS is_primary
		FROM location_history lh
		INNER JOIN group_members gm ON gm.user_id = lh.user_id AND gm.group_id = $1
		INNER JOIN users u ON u.id = lh.user_id
		LEFT JOIN devices d ON d.id = lh.device_id
		WHERE lh.recorded_at > NOW() - INTERVAL '1 hour'
		ORDER BY lh.device_id, lh.recorded_at DESC`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LocationRecord
	for rows.Next() {
		var rec LocationRecord
		if err := rows.Scan(
			&rec.UserID, &rec.DeviceID,
			&rec.Lat, &rec.Lng,
			&rec.Altitude, &rec.Heading, &rec.Speed, &rec.Accuracy,
			&rec.RecordedAt,
			&rec.Username, &rec.DisplayName,
			&rec.DeviceName,
			&rec.IsPrimary,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// GetGroupHistory returns all location records for a group within a time range,
// ordered by user and then time ascending (for track rendering).
func (r *LocationRepository) GetGroupHistory(ctx context.Context, groupID uuid.UUID, from, to time.Time) ([]LocationRecord, error) {
	query := `
		SELECT lh.user_id, lh.device_id,
		       ST_Y(lh.location) AS lat, ST_X(lh.location) AS lng,
		       lh.altitude, lh.heading, lh.speed, lh.accuracy,
		       lh.recorded_at,
		       u.username, u.display_name,
		       d.name AS device_name
		FROM location_history lh
		INNER JOIN group_members gm ON gm.user_id = lh.user_id AND gm.group_id = $1
		INNER JOIN users u ON u.id = lh.user_id
		LEFT JOIN devices d ON d.id = lh.device_id
		WHERE lh.recorded_at >= $2 AND lh.recorded_at <= $3
		ORDER BY lh.user_id, lh.recorded_at ASC`

	rows, err := r.pool.Query(ctx, query, groupID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LocationRecord
	for rows.Next() {
		var rec LocationRecord
		if err := rows.Scan(
			&rec.UserID, &rec.DeviceID,
			&rec.Lat, &rec.Lng,
			&rec.Altitude, &rec.Heading, &rec.Speed, &rec.Accuracy,
			&rec.RecordedAt,
			&rec.Username, &rec.DisplayName,
			&rec.DeviceName,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// GetUserHistory returns all location records for a single user within a time range.
// If deviceID is non-nil, results are filtered to that specific device.
func (r *LocationRepository) GetUserHistory(ctx context.Context, userID uuid.UUID, from, to time.Time, deviceID *uuid.UUID) ([]LocationRecord, error) {
	query := `
		SELECT lh.user_id, lh.device_id,
		       ST_Y(lh.location) AS lat, ST_X(lh.location) AS lng,
		       lh.altitude, lh.heading, lh.speed, lh.accuracy,
		       lh.recorded_at,
		       u.username, u.display_name,
		       d.name AS device_name
		FROM location_history lh
		INNER JOIN users u ON u.id = lh.user_id
		LEFT JOIN devices d ON d.id = lh.device_id
		WHERE lh.user_id = $1 AND lh.recorded_at >= $2 AND lh.recorded_at <= $3`

	args := []any{userID, from, to}
	if deviceID != nil {
		query += ` AND lh.device_id = $4`
		args = append(args, *deviceID)
	}
	query += ` ORDER BY lh.recorded_at ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LocationRecord
	for rows.Next() {
		var rec LocationRecord
		if err := rows.Scan(
			&rec.UserID, &rec.DeviceID,
			&rec.Lat, &rec.Lng,
			&rec.Altitude, &rec.Heading, &rec.Speed, &rec.Accuracy,
			&rec.RecordedAt,
			&rec.Username, &rec.DisplayName,
			&rec.DeviceName,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// GetVisibleHistory returns location history for all users visible to the caller
// (i.e. users who share at least one group with the caller) within a time range.
func (r *LocationRepository) GetVisibleHistory(ctx context.Context, callerID uuid.UUID, from, to time.Time) ([]LocationRecord, error) {
	query := `
		SELECT lh.user_id, lh.device_id,
		       ST_Y(lh.location) AS lat, ST_X(lh.location) AS lng,
		       lh.altitude, lh.heading, lh.speed, lh.accuracy,
		       lh.recorded_at,
		       u.username, u.display_name,
		       d.name AS device_name
		FROM location_history lh
		INNER JOIN users u ON u.id = lh.user_id
		LEFT JOIN devices d ON d.id = lh.device_id
		WHERE lh.user_id IN (
			SELECT DISTINCT gm2.user_id
			FROM group_members gm1
			INNER JOIN group_members gm2 ON gm1.group_id = gm2.group_id
			WHERE gm1.user_id = $1
		)
		AND lh.recorded_at >= $2 AND lh.recorded_at <= $3
		ORDER BY lh.user_id, lh.recorded_at ASC`

	rows, err := r.pool.Query(ctx, query, callerID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LocationRecord
	for rows.Next() {
		var rec LocationRecord
		if err := rows.Scan(
			&rec.UserID, &rec.DeviceID,
			&rec.Lat, &rec.Lng,
			&rec.Altitude, &rec.Heading, &rec.Speed, &rec.Accuracy,
			&rec.RecordedAt,
			&rec.Username, &rec.DisplayName,
			&rec.DeviceName,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// GetAllHistory returns location history for ALL users within a time range (admin only).
func (r *LocationRepository) GetAllHistory(ctx context.Context, from, to time.Time) ([]LocationRecord, error) {
	query := `
		SELECT lh.user_id, lh.device_id,
		       ST_Y(lh.location) AS lat, ST_X(lh.location) AS lng,
		       lh.altitude, lh.heading, lh.speed, lh.accuracy,
		       lh.recorded_at,
		       u.username, u.display_name,
		       d.name AS device_name
		FROM location_history lh
		INNER JOIN users u ON u.id = lh.user_id
		LEFT JOIN devices d ON d.id = lh.device_id
		WHERE lh.recorded_at >= $1 AND lh.recorded_at <= $2
		ORDER BY lh.user_id, lh.recorded_at ASC`

	rows, err := r.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LocationRecord
	for rows.Next() {
		var rec LocationRecord
		if err := rows.Scan(
			&rec.UserID, &rec.DeviceID,
			&rec.Lat, &rec.Lng,
			&rec.Altitude, &rec.Heading, &rec.Speed, &rec.Accuracy,
			&rec.RecordedAt,
			&rec.Username, &rec.DisplayName,
			&rec.DeviceName,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// UsersShareGroup returns true if the two users share at least one group.
func (r *LocationRepository) UsersShareGroup(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM group_members gm1
			INNER JOIN group_members gm2 ON gm1.group_id = gm2.group_id
			WHERE gm1.user_id = $1 AND gm2.user_id = $2
		)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userA, userB).Scan(&exists)
	return exists, err
}

// GetLatestByUser returns the most recent location for each of a specific user's
// devices within the past hour. Used to send a self-snapshot on WebSocket connect
// so the user sees their own other devices immediately, regardless of group membership.
func (r *LocationRepository) GetLatestByUser(ctx context.Context, userID uuid.UUID) ([]LocationRecord, error) {
	query := `
		SELECT DISTINCT ON (lh.device_id)
			lh.user_id, lh.device_id,
			ST_Y(lh.location) AS lat, ST_X(lh.location) AS lng,
			lh.altitude, lh.heading, lh.speed, lh.accuracy,
			lh.recorded_at,
			u.username, u.display_name,
			d.name AS device_name,
			COALESCE(d.is_primary, false) AS is_primary
		FROM location_history lh
		INNER JOIN users u ON u.id = lh.user_id
		LEFT JOIN devices d ON d.id = lh.device_id
		WHERE lh.user_id = $1
		  AND lh.recorded_at > NOW() - INTERVAL '1 hour'
		ORDER BY lh.device_id, lh.recorded_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LocationRecord
	for rows.Next() {
		var rec LocationRecord
		if err := rows.Scan(
			&rec.UserID, &rec.DeviceID,
			&rec.Lat, &rec.Lng,
			&rec.Altitude, &rec.Heading, &rec.Speed, &rec.Accuracy,
			&rec.RecordedAt,
			&rec.Username, &rec.DisplayName,
			&rec.DeviceName,
			&rec.IsPrimary,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// GetAllLatest returns the most recent location for each device across all groups.
// Used by admin to see all device positions.
func (r *LocationRepository) GetAllLatest(ctx context.Context) ([]LocationRecord, error) {
	query := `
		SELECT DISTINCT ON (lh.device_id)
			lh.user_id, lh.device_id,
			ST_Y(lh.location) AS lat, ST_X(lh.location) AS lng,
			lh.altitude, lh.heading, lh.speed, lh.accuracy,
			lh.recorded_at,
			u.username, u.display_name,
			d.name AS device_name,
			COALESCE(d.is_primary, false) AS is_primary
		FROM location_history lh
		INNER JOIN users u ON u.id = lh.user_id
		LEFT JOIN devices d ON d.id = lh.device_id
		WHERE lh.recorded_at > NOW() - INTERVAL '1 hour'
		ORDER BY lh.device_id, lh.recorded_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LocationRecord
	for rows.Next() {
		var rec LocationRecord
		if err := rows.Scan(
			&rec.UserID, &rec.DeviceID,
			&rec.Lat, &rec.Lng,
			&rec.Altitude, &rec.Heading, &rec.Speed, &rec.Accuracy,
			&rec.RecordedAt,
			&rec.Username, &rec.DisplayName,
			&rec.DeviceName,
			&rec.IsPrimary,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}
