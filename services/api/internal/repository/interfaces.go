// Package repository defines data‐access interfaces for all domain entities.
// Concrete implementations backed by PostgreSQL live alongside these
// interfaces in the same package. Service and WebSocket layers accept the
// interfaces so that tests can substitute lightweight in‐memory mocks.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
)

// ---------------------------------------------------------------------------
// UserRepo
// ---------------------------------------------------------------------------

// UserRepo abstracts user persistence.
type UserRepo interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context, page, pageSize int) ([]model.User, int, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountAdmins(ctx context.Context) (int, error)
	SetMFAEnabled(ctx context.Context, id uuid.UUID, enabled bool) error
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// ---------------------------------------------------------------------------
// TokenRepo
// ---------------------------------------------------------------------------

// TokenRepo abstracts refresh‐token persistence.
type TokenRepo interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	GetByHash(ctx context.Context, hash string) (*model.RefreshToken, error)
	DeleteByHash(ctx context.Context, hash string) error
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// ---------------------------------------------------------------------------
// GroupRepo
// ---------------------------------------------------------------------------

// GroupRepo abstracts group and group‐member persistence.
type GroupRepo interface {
	Create(ctx context.Context, group *model.Group) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Group, error)
	List(ctx context.Context, page, pageSize int) ([]model.Group, []int, int, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Group, []int, error)
	Update(ctx context.Context, group *model.Group) error
	UpdateMarker(ctx context.Context, id uuid.UUID, markerIcon, markerColor string) (*model.Group, error)
	Delete(ctx context.Context, id uuid.UUID) error
	MemberCount(ctx context.Context, groupID uuid.UUID) (int, error)

	// Members
	AddMember(ctx context.Context, member *model.GroupMember) error
	GetMember(ctx context.Context, groupID, userID uuid.UUID) (*model.GroupMember, error)
	GetMemberByID(ctx context.Context, memberID uuid.UUID) (*model.GroupMember, error)
	ListMembers(ctx context.Context, groupID uuid.UUID) ([]model.GroupMemberWithUser, error)
	UpdateMember(ctx context.Context, member *model.GroupMember) error
	RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error
}

// ---------------------------------------------------------------------------
// LocationRepo
// ---------------------------------------------------------------------------

// LocationRepo abstracts location history persistence.
type LocationRepo interface {
	Create(ctx context.Context, userID, deviceID uuid.UUID, lat, lng float64, altitude, heading, speed, accuracy *float64) error
	UpdateDeviceLocation(ctx context.Context, deviceID uuid.UUID, lat, lng float64) error
	GetLatestByGroup(ctx context.Context, groupID uuid.UUID) ([]LocationRecord, error)
	GetGroupHistory(ctx context.Context, groupID uuid.UUID, from, to time.Time) ([]LocationRecord, error)
	GetUserHistory(ctx context.Context, userID uuid.UUID, from, to time.Time, deviceID *uuid.UUID) ([]LocationRecord, error)
	GetVisibleHistory(ctx context.Context, callerID uuid.UUID, from, to time.Time) ([]LocationRecord, error)
	GetAllHistory(ctx context.Context, from, to time.Time) ([]LocationRecord, error)
	UsersShareGroup(ctx context.Context, userA, userB uuid.UUID) (bool, error)
	GetAllLatest(ctx context.Context) ([]LocationRecord, error)
}

// ---------------------------------------------------------------------------
// MessageRepo
// ---------------------------------------------------------------------------

// MessageRepo abstracts message and attachment persistence.
type MessageRepo interface {
	Create(ctx context.Context, msg *model.Message) error
	CreateAttachment(ctx context.Context, att *model.Attachment) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.MessageWithUser, error)
	ListByGroup(ctx context.Context, groupID uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error)
	ListDirect(ctx context.Context, userA, userB uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetAttachmentByID(ctx context.Context, id uuid.UUID) (*model.Attachment, error)
	GetAttachmentObjectKeys(ctx context.Context, messageID uuid.UUID) ([]string, error)
	ListDMPartners(ctx context.Context, userID uuid.UUID) ([]DMPartner, error)
}

// ---------------------------------------------------------------------------
// DrawingRepo
// ---------------------------------------------------------------------------

// DrawingRepo abstracts drawing persistence.
type DrawingRepo interface {
	Create(ctx context.Context, d *model.Drawing) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]model.DrawingWithUser, error)
	ListSharedWithUser(ctx context.Context, userID uuid.UUID) ([]model.DrawingWithUser, error)
	Update(ctx context.Context, d *model.Drawing) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetShareTargets(ctx context.Context, drawingID uuid.UUID) (groupIDs []uuid.UUID, userIDs []uuid.UUID, err error)
	ListShares(ctx context.Context, drawingID uuid.UUID) ([]model.DrawingShareInfo, error)
	RevokeShare(ctx context.Context, messageID uuid.UUID) error
}

// ---------------------------------------------------------------------------
// AuditRepo
// ---------------------------------------------------------------------------

// AuditRepo abstracts audit‐log persistence.
type AuditRepo interface {
	Create(ctx context.Context, log *model.AuditLog) error
	ListByUser(ctx context.Context, userID uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error)
	ListByGroup(ctx context.Context, groupID uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error)
	ListAll(ctx context.Context, f model.AuditFilters) ([]model.AuditLogWithUser, int, error)
}

// ---------------------------------------------------------------------------
// CotRepo
// ---------------------------------------------------------------------------

// CotRepo abstracts CoT event persistence.
type CotRepo interface {
	Create(ctx context.Context, evt *model.CotEvent) error
	List(ctx context.Context, f model.CotEventFilters) ([]model.CotEvent, int, error)
	GetLatestByUID(ctx context.Context, eventUID string) (*model.CotEvent, error)
}

// ---------------------------------------------------------------------------
// DeviceRepo
// ---------------------------------------------------------------------------

// DeviceRepo abstracts device persistence.
type DeviceRepo interface {
	Create(ctx context.Context, device *model.Device) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Device, error)
	Update(ctx context.Context, device *model.Device) error
	TouchLastSeen(ctx context.Context, id uuid.UUID, userAgent *string) error
	GetByDeviceUID(ctx context.Context, deviceUID string) (*model.Device, error)
	FindSingleByUserAgent(ctx context.Context, userID uuid.UUID, deviceType, userAgent string) (*model.Device, error)
	SetPrimary(ctx context.Context, userID, deviceID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---------------------------------------------------------------------------
// MapConfigRepo
// ---------------------------------------------------------------------------

// MapConfigRepo abstracts map configuration persistence.
type MapConfigRepo interface {
	Create(ctx context.Context, mc *model.MapConfig) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.MapConfig, error)
	GetDefault(ctx context.Context) (*model.MapConfig, error)
	List(ctx context.Context) ([]model.MapConfig, error)
	Update(ctx context.Context, mc *model.MapConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
	ClearDefault(ctx context.Context) error
	CountBuiltin(ctx context.Context) (int64, error)
}

// ---------------------------------------------------------------------------
// TerrainConfigRepo
// ---------------------------------------------------------------------------

// TerrainConfigRepo abstracts terrain configuration persistence.
type TerrainConfigRepo interface {
	Create(ctx context.Context, tc *model.TerrainConfig) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.TerrainConfig, error)
	GetDefault(ctx context.Context) (*model.TerrainConfig, error)
	List(ctx context.Context) ([]model.TerrainConfig, error)
	Update(ctx context.Context, tc *model.TerrainConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
	ClearDefault(ctx context.Context) error
	CountBuiltin(ctx context.Context) (int64, error)
}

// ---------------------------------------------------------------------------
// ServerSettingsRepo
// ---------------------------------------------------------------------------

// ServerSettingsRepo abstracts server‐settings persistence.
type ServerSettingsRepo interface {
	Get(ctx context.Context, key string) (*model.ServerSetting, error)
	Set(ctx context.Context, key, value string) error
	GetAll(ctx context.Context) ([]model.ServerSetting, error)
}

// ---------------------------------------------------------------------------
// MFARepo
// ---------------------------------------------------------------------------

// MFARepo abstracts MFA persistence (TOTP, WebAuthn, recovery codes).
type MFARepo interface {
	// TOTP
	CreateTOTP(ctx context.Context, m *model.TOTPMethod) error
	GetTOTPByID(ctx context.Context, id uuid.UUID) (*model.TOTPMethod, error)
	ListTOTPByUser(ctx context.Context, userID uuid.UUID) ([]model.TOTPMethod, error)
	VerifyTOTP(ctx context.Context, id uuid.UUID) error
	TouchTOTP(ctx context.Context, id uuid.UUID) error
	DeleteTOTP(ctx context.Context, id uuid.UUID) error
	DeleteAllTOTPForUser(ctx context.Context, userID uuid.UUID) error

	// WebAuthn
	CreateWebAuthn(ctx context.Context, c *model.WebAuthnCredential) error
	GetWebAuthnByID(ctx context.Context, id uuid.UUID) (*model.WebAuthnCredential, error)
	GetWebAuthnByCredentialID(ctx context.Context, credentialID []byte) (*model.WebAuthnCredential, error)
	ListWebAuthnByUser(ctx context.Context, userID uuid.UUID) ([]model.WebAuthnCredential, error)
	ListPasswordlessCredentials(ctx context.Context) ([]model.WebAuthnCredential, error)
	UpdateWebAuthnSignCount(ctx context.Context, id uuid.UUID, signCount int64, backupState bool) error
	UpdateWebAuthnPasswordless(ctx context.Context, id uuid.UUID, enabled bool) error
	DeleteWebAuthn(ctx context.Context, id uuid.UUID) error
	DeleteAllWebAuthnForUser(ctx context.Context, userID uuid.UUID) error

	// Recovery codes
	CreateRecoveryCodes(ctx context.Context, userID uuid.UUID, hashes []string) error
	ListUnusedRecoveryCodes(ctx context.Context, userID uuid.UUID) ([]model.RecoveryCode, error)
	MarkRecoveryCodeUsed(ctx context.Context, id uuid.UUID) error
	DeleteAllRecoveryCodesForUser(ctx context.Context, userID uuid.UUID) error
	CountUnusedRecoveryCodes(ctx context.Context, userID uuid.UUID) (int, error)

	// Aggregate helpers
	CountVerifiedMethods(ctx context.Context, userID uuid.UUID) (int, error)
	HasVerifiedTOTP(ctx context.Context, userID uuid.UUID) (bool, error)
	HasWebAuthn(ctx context.Context, userID uuid.UUID) (bool, error)
}

// ---------------------------------------------------------------------------
// APITokenRepo
// ---------------------------------------------------------------------------

// APITokenRepo abstracts API token persistence.
type APITokenRepo interface {
	Create(ctx context.Context, token *model.APIToken) error
	GetByTokenHash(ctx context.Context, hash string) (*model.APIToken, *model.User, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error)
	Delete(ctx context.Context, userID, tokenID uuid.UUID) error
	TouchLastUsed(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// ---------------------------------------------------------------------------
// Compile-time interface satisfaction checks
// ---------------------------------------------------------------------------

var (
	_ UserRepo           = (*UserRepository)(nil)
	_ TokenRepo          = (*TokenRepository)(nil)
	_ GroupRepo          = (*GroupRepository)(nil)
	_ LocationRepo       = (*LocationRepository)(nil)
	_ MessageRepo        = (*MessageRepository)(nil)
	_ DrawingRepo        = (*DrawingRepository)(nil)
	_ AuditRepo          = (*AuditRepository)(nil)
	_ CotRepo            = (*CotRepository)(nil)
	_ DeviceRepo         = (*DeviceRepository)(nil)
	_ MapConfigRepo      = (*MapConfigRepository)(nil)
	_ TerrainConfigRepo  = (*TerrainConfigRepository)(nil)
	_ ServerSettingsRepo = (*ServerSettingsRepository)(nil)
	_ MFARepo            = (*MFARepository)(nil)
	_ APITokenRepo       = (*APITokenRepository)(nil)
)
