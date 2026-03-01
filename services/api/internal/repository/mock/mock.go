// Package mock provides in-memory mock implementations of repository interfaces
// for unit testing. Each mock struct uses function fields so tests can override
// only the methods they care about.
package mock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// ---------------------------------------------------------------------------
// UserRepo
// ---------------------------------------------------------------------------

// UserRepo is a mock implementation of repository.UserRepo.
type UserRepo struct {
	CreateFn           func(ctx context.Context, user *model.User) error
	GetByIDFn          func(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByUsernameFn    func(ctx context.Context, username string) (*model.User, error)
	GetByEmailFn       func(ctx context.Context, email string) (*model.User, error)
	ListFn             func(ctx context.Context, page, pageSize int) ([]model.User, int, error)
	UpdateFn           func(ctx context.Context, user *model.User) error
	DeleteFn           func(ctx context.Context, id uuid.UUID) error
	CountAdminsFn      func(ctx context.Context) (int, error)
	SetMFAEnabledFn    func(ctx context.Context, id uuid.UUID, enabled bool) error
	ExistsByUsernameFn func(ctx context.Context, username string) (bool, error)
	ExistsByEmailFn    func(ctx context.Context, email string) (bool, error)
}

func (m *UserRepo) Create(ctx context.Context, user *model.User) error {
	return m.CreateFn(ctx, user)
}
func (m *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return m.GetByUsernameFn(ctx, username)
}
func (m *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return m.GetByEmailFn(ctx, email)
}
func (m *UserRepo) List(ctx context.Context, page, pageSize int) ([]model.User, int, error) {
	return m.ListFn(ctx, page, pageSize)
}
func (m *UserRepo) Update(ctx context.Context, user *model.User) error {
	return m.UpdateFn(ctx, user)
}
func (m *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *UserRepo) CountAdmins(ctx context.Context) (int, error) {
	return m.CountAdminsFn(ctx)
}
func (m *UserRepo) SetMFAEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	return m.SetMFAEnabledFn(ctx, id, enabled)
}
func (m *UserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return m.ExistsByUsernameFn(ctx, username)
}
func (m *UserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return m.ExistsByEmailFn(ctx, email)
}

var _ repository.UserRepo = (*UserRepo)(nil)

// ---------------------------------------------------------------------------
// TokenRepo
// ---------------------------------------------------------------------------

// TokenRepo is a mock implementation of repository.TokenRepo.
type TokenRepo struct {
	CreateFn           func(ctx context.Context, token *model.RefreshToken) error
	GetByHashFn        func(ctx context.Context, hash string) (*model.RefreshToken, error)
	DeleteByHashFn     func(ctx context.Context, hash string) error
	DeleteAllForUserFn func(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredFn    func(ctx context.Context) (int64, error)
}

func (m *TokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	return m.CreateFn(ctx, token)
}
func (m *TokenRepo) GetByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	return m.GetByHashFn(ctx, hash)
}
func (m *TokenRepo) DeleteByHash(ctx context.Context, hash string) error {
	return m.DeleteByHashFn(ctx, hash)
}
func (m *TokenRepo) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	return m.DeleteAllForUserFn(ctx, userID)
}
func (m *TokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	return m.DeleteExpiredFn(ctx)
}

var _ repository.TokenRepo = (*TokenRepo)(nil)

// ---------------------------------------------------------------------------
// GroupRepo
// ---------------------------------------------------------------------------

// GroupRepo is a mock implementation of repository.GroupRepo.
type GroupRepo struct {
	CreateFn        func(ctx context.Context, group *model.Group) error
	GetByIDFn       func(ctx context.Context, id uuid.UUID) (*model.Group, error)
	ListFn          func(ctx context.Context, page, pageSize int) ([]model.Group, []int, int, error)
	ListByUserIDFn  func(ctx context.Context, userID uuid.UUID) ([]model.Group, []int, error)
	UpdateFn        func(ctx context.Context, group *model.Group) error
	UpdateMarkerFn  func(ctx context.Context, id uuid.UUID, markerIcon, markerColor string) (*model.Group, error)
	DeleteFn        func(ctx context.Context, id uuid.UUID) error
	MemberCountFn   func(ctx context.Context, groupID uuid.UUID) (int, error)
	AddMemberFn     func(ctx context.Context, member *model.GroupMember) error
	GetMemberFn     func(ctx context.Context, groupID, userID uuid.UUID) (*model.GroupMember, error)
	GetMemberByIDFn func(ctx context.Context, memberID uuid.UUID) (*model.GroupMember, error)
	ListMembersFn              func(ctx context.Context, groupID uuid.UUID) ([]model.GroupMemberWithUser, error)
	ListMembershipsByUserIDFn  func(ctx context.Context, userID uuid.UUID) ([]model.GroupMember, error)
	UpdateMemberFn             func(ctx context.Context, member *model.GroupMember) error
	RemoveMemberFn             func(ctx context.Context, groupID, userID uuid.UUID) error
}

func (m *GroupRepo) Create(ctx context.Context, group *model.Group) error {
	return m.CreateFn(ctx, group)
}
func (m *GroupRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Group, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *GroupRepo) List(ctx context.Context, page, pageSize int) ([]model.Group, []int, int, error) {
	return m.ListFn(ctx, page, pageSize)
}
func (m *GroupRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Group, []int, error) {
	return m.ListByUserIDFn(ctx, userID)
}
func (m *GroupRepo) Update(ctx context.Context, group *model.Group) error {
	return m.UpdateFn(ctx, group)
}
func (m *GroupRepo) UpdateMarker(ctx context.Context, id uuid.UUID, markerIcon, markerColor string) (*model.Group, error) {
	return m.UpdateMarkerFn(ctx, id, markerIcon, markerColor)
}
func (m *GroupRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *GroupRepo) MemberCount(ctx context.Context, groupID uuid.UUID) (int, error) {
	return m.MemberCountFn(ctx, groupID)
}
func (m *GroupRepo) AddMember(ctx context.Context, member *model.GroupMember) error {
	return m.AddMemberFn(ctx, member)
}
func (m *GroupRepo) GetMember(ctx context.Context, groupID, userID uuid.UUID) (*model.GroupMember, error) {
	return m.GetMemberFn(ctx, groupID, userID)
}
func (m *GroupRepo) GetMemberByID(ctx context.Context, memberID uuid.UUID) (*model.GroupMember, error) {
	return m.GetMemberByIDFn(ctx, memberID)
}
func (m *GroupRepo) ListMembers(ctx context.Context, groupID uuid.UUID) ([]model.GroupMemberWithUser, error) {
	return m.ListMembersFn(ctx, groupID)
}
func (m *GroupRepo) ListMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]model.GroupMember, error) {
	return m.ListMembershipsByUserIDFn(ctx, userID)
}
func (m *GroupRepo) UpdateMember(ctx context.Context, member *model.GroupMember) error {
	return m.UpdateMemberFn(ctx, member)
}
func (m *GroupRepo) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	return m.RemoveMemberFn(ctx, groupID, userID)
}

var _ repository.GroupRepo = (*GroupRepo)(nil)

// ---------------------------------------------------------------------------
// LocationRepo
// ---------------------------------------------------------------------------

// LocationRepo is a mock implementation of repository.LocationRepo.
type LocationRepo struct {
	CreateFn               func(ctx context.Context, userID, deviceID uuid.UUID, lat, lng float64, altitude, heading, speed, accuracy *float64) error
	UpdateDeviceLocationFn func(ctx context.Context, deviceID uuid.UUID, lat, lng float64) error
	GetLatestByGroupFn     func(ctx context.Context, groupID uuid.UUID) ([]repository.LocationRecord, error)
	GetGroupHistoryFn      func(ctx context.Context, groupID uuid.UUID, from, to time.Time) ([]repository.LocationRecord, error)
	GetUserHistoryFn       func(ctx context.Context, userID uuid.UUID, from, to time.Time, deviceID *uuid.UUID) ([]repository.LocationRecord, error)
	GetVisibleHistoryFn    func(ctx context.Context, callerID uuid.UUID, from, to time.Time) ([]repository.LocationRecord, error)
	GetAllHistoryFn        func(ctx context.Context, from, to time.Time) ([]repository.LocationRecord, error)
	UsersShareGroupFn      func(ctx context.Context, userA, userB uuid.UUID) (bool, error)
	GetLatestByUserFn      func(ctx context.Context, userID uuid.UUID) ([]repository.LocationRecord, error)
	GetAllLatestFn         func(ctx context.Context) ([]repository.LocationRecord, error)
}

func (m *LocationRepo) Create(ctx context.Context, userID, deviceID uuid.UUID, lat, lng float64, altitude, heading, speed, accuracy *float64) error {
	return m.CreateFn(ctx, userID, deviceID, lat, lng, altitude, heading, speed, accuracy)
}
func (m *LocationRepo) UpdateDeviceLocation(ctx context.Context, deviceID uuid.UUID, lat, lng float64) error {
	return m.UpdateDeviceLocationFn(ctx, deviceID, lat, lng)
}
func (m *LocationRepo) GetLatestByGroup(ctx context.Context, groupID uuid.UUID) ([]repository.LocationRecord, error) {
	return m.GetLatestByGroupFn(ctx, groupID)
}
func (m *LocationRepo) GetGroupHistory(ctx context.Context, groupID uuid.UUID, from, to time.Time) ([]repository.LocationRecord, error) {
	return m.GetGroupHistoryFn(ctx, groupID, from, to)
}
func (m *LocationRepo) GetUserHistory(ctx context.Context, userID uuid.UUID, from, to time.Time, deviceID *uuid.UUID) ([]repository.LocationRecord, error) {
	return m.GetUserHistoryFn(ctx, userID, from, to, deviceID)
}
func (m *LocationRepo) GetVisibleHistory(ctx context.Context, callerID uuid.UUID, from, to time.Time) ([]repository.LocationRecord, error) {
	return m.GetVisibleHistoryFn(ctx, callerID, from, to)
}
func (m *LocationRepo) GetAllHistory(ctx context.Context, from, to time.Time) ([]repository.LocationRecord, error) {
	return m.GetAllHistoryFn(ctx, from, to)
}
func (m *LocationRepo) UsersShareGroup(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	return m.UsersShareGroupFn(ctx, userA, userB)
}
func (m *LocationRepo) GetLatestByUser(ctx context.Context, userID uuid.UUID) ([]repository.LocationRecord, error) {
	return m.GetLatestByUserFn(ctx, userID)
}
func (m *LocationRepo) GetAllLatest(ctx context.Context) ([]repository.LocationRecord, error) {
	return m.GetAllLatestFn(ctx)
}

var _ repository.LocationRepo = (*LocationRepo)(nil)

// ---------------------------------------------------------------------------
// MessageRepo
// ---------------------------------------------------------------------------

// MessageRepo is a mock implementation of repository.MessageRepo.
type MessageRepo struct {
	CreateFn                  func(ctx context.Context, msg *model.Message) error
	CreateAttachmentFn        func(ctx context.Context, att *model.Attachment) error
	GetByIDFn                 func(ctx context.Context, id uuid.UUID) (*model.MessageWithUser, error)
	ListByGroupFn             func(ctx context.Context, groupID uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error)
	ListDirectFn              func(ctx context.Context, userA, userB uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error)
	DeleteFn                  func(ctx context.Context, id uuid.UUID) error
	GetAttachmentByIDFn       func(ctx context.Context, id uuid.UUID) (*model.Attachment, error)
	GetAttachmentObjectKeysFn func(ctx context.Context, messageID uuid.UUID) ([]string, error)
	ListDMPartnersFn          func(ctx context.Context, userID uuid.UUID) ([]repository.DMPartner, error)
}

func (m *MessageRepo) Create(ctx context.Context, msg *model.Message) error {
	return m.CreateFn(ctx, msg)
}
func (m *MessageRepo) CreateAttachment(ctx context.Context, att *model.Attachment) error {
	return m.CreateAttachmentFn(ctx, att)
}
func (m *MessageRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.MessageWithUser, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MessageRepo) ListByGroup(ctx context.Context, groupID uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error) {
	return m.ListByGroupFn(ctx, groupID, before, limit)
}
func (m *MessageRepo) ListDirect(ctx context.Context, userA, userB uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error) {
	return m.ListDirectFn(ctx, userA, userB, before, limit)
}
func (m *MessageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *MessageRepo) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*model.Attachment, error) {
	return m.GetAttachmentByIDFn(ctx, id)
}
func (m *MessageRepo) GetAttachmentObjectKeys(ctx context.Context, messageID uuid.UUID) ([]string, error) {
	return m.GetAttachmentObjectKeysFn(ctx, messageID)
}
func (m *MessageRepo) ListDMPartners(ctx context.Context, userID uuid.UUID) ([]repository.DMPartner, error) {
	return m.ListDMPartnersFn(ctx, userID)
}

var _ repository.MessageRepo = (*MessageRepo)(nil)

// ---------------------------------------------------------------------------
// DrawingRepo
// ---------------------------------------------------------------------------

// DrawingRepo is a mock implementation of repository.DrawingRepo.
type DrawingRepo struct {
	CreateFn             func(ctx context.Context, d *model.Drawing) error
	GetByIDFn            func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error)
	ListByOwnerFn        func(ctx context.Context, ownerID uuid.UUID) ([]model.DrawingWithUser, error)
	ListSharedWithUserFn func(ctx context.Context, userID uuid.UUID) ([]model.DrawingWithUser, error)
	UpdateFn             func(ctx context.Context, d *model.Drawing) error
	DeleteFn             func(ctx context.Context, id uuid.UUID) error
	GetShareTargetsFn    func(ctx context.Context, drawingID uuid.UUID) (groupIDs []uuid.UUID, userIDs []uuid.UUID, err error)
	ListSharesFn         func(ctx context.Context, drawingID uuid.UUID) ([]model.DrawingShareInfo, error)
	RevokeShareFn        func(ctx context.Context, messageID uuid.UUID) error
}

func (m *DrawingRepo) Create(ctx context.Context, d *model.Drawing) error {
	return m.CreateFn(ctx, d)
}
func (m *DrawingRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *DrawingRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]model.DrawingWithUser, error) {
	return m.ListByOwnerFn(ctx, ownerID)
}
func (m *DrawingRepo) ListSharedWithUser(ctx context.Context, userID uuid.UUID) ([]model.DrawingWithUser, error) {
	return m.ListSharedWithUserFn(ctx, userID)
}
func (m *DrawingRepo) Update(ctx context.Context, d *model.Drawing) error {
	return m.UpdateFn(ctx, d)
}
func (m *DrawingRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *DrawingRepo) GetShareTargets(ctx context.Context, drawingID uuid.UUID) (groupIDs []uuid.UUID, userIDs []uuid.UUID, err error) {
	return m.GetShareTargetsFn(ctx, drawingID)
}
func (m *DrawingRepo) ListShares(ctx context.Context, drawingID uuid.UUID) ([]model.DrawingShareInfo, error) {
	return m.ListSharesFn(ctx, drawingID)
}
func (m *DrawingRepo) RevokeShare(ctx context.Context, messageID uuid.UUID) error {
	return m.RevokeShareFn(ctx, messageID)
}

var _ repository.DrawingRepo = (*DrawingRepo)(nil)

// ---------------------------------------------------------------------------
// AuditRepo
// ---------------------------------------------------------------------------

// AuditRepo is a mock implementation of repository.AuditRepo.
type AuditRepo struct {
	CreateFn      func(ctx context.Context, log *model.AuditLog) error
	ListByUserFn  func(ctx context.Context, userID uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error)
	ListByGroupFn func(ctx context.Context, groupID uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error)
	ListAllFn     func(ctx context.Context, f model.AuditFilters) ([]model.AuditLogWithUser, int, error)
}

func (m *AuditRepo) Create(ctx context.Context, log *model.AuditLog) error {
	return m.CreateFn(ctx, log)
}
func (m *AuditRepo) ListByUser(ctx context.Context, userID uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
	return m.ListByUserFn(ctx, userID, f)
}
func (m *AuditRepo) ListByGroup(ctx context.Context, groupID uuid.UUID, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
	return m.ListByGroupFn(ctx, groupID, f)
}
func (m *AuditRepo) ListAll(ctx context.Context, f model.AuditFilters) ([]model.AuditLogWithUser, int, error) {
	return m.ListAllFn(ctx, f)
}

var _ repository.AuditRepo = (*AuditRepo)(nil)

// ---------------------------------------------------------------------------
// CotRepo
// ---------------------------------------------------------------------------

// CotRepo is a mock implementation of repository.CotRepo.
type CotRepo struct {
	CreateFn         func(ctx context.Context, evt *model.CotEvent) error
	ListFn           func(ctx context.Context, f model.CotEventFilters) ([]model.CotEvent, int, error)
	GetLatestByUIDFn func(ctx context.Context, eventUID string) (*model.CotEvent, error)
}

func (m *CotRepo) Create(ctx context.Context, evt *model.CotEvent) error {
	return m.CreateFn(ctx, evt)
}
func (m *CotRepo) List(ctx context.Context, f model.CotEventFilters) ([]model.CotEvent, int, error) {
	return m.ListFn(ctx, f)
}
func (m *CotRepo) GetLatestByUID(ctx context.Context, eventUID string) (*model.CotEvent, error) {
	return m.GetLatestByUIDFn(ctx, eventUID)
}

var _ repository.CotRepo = (*CotRepo)(nil)

// ---------------------------------------------------------------------------
// DeviceRepo
// ---------------------------------------------------------------------------

// DeviceRepo is a mock implementation of repository.DeviceRepo.
type DeviceRepo struct {
	CreateFn                func(ctx context.Context, device *model.Device) error
	GetByIDFn               func(ctx context.Context, id uuid.UUID) (*model.Device, error)
	ListByUserIDFn          func(ctx context.Context, userID uuid.UUID) ([]model.Device, error)
	UpdateFn                func(ctx context.Context, device *model.Device) error
	TouchLastSeenFn         func(ctx context.Context, id uuid.UUID, userAgent *string, appVersion *string) error
	GetByDeviceUIDFn        func(ctx context.Context, deviceUID string) (*model.Device, error)
	FindSingleByUserAgentFn func(ctx context.Context, userID uuid.UUID, deviceType, userAgent string) (*model.Device, error)
	SetPrimaryFn            func(ctx context.Context, userID, deviceID uuid.UUID) error
	DeleteFn                func(ctx context.Context, id uuid.UUID) error
}

func (m *DeviceRepo) Create(ctx context.Context, device *model.Device) error {
	return m.CreateFn(ctx, device)
}
func (m *DeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *DeviceRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Device, error) {
	return m.ListByUserIDFn(ctx, userID)
}
func (m *DeviceRepo) Update(ctx context.Context, device *model.Device) error {
	return m.UpdateFn(ctx, device)
}
func (m *DeviceRepo) TouchLastSeen(ctx context.Context, id uuid.UUID, userAgent *string, appVersion *string) error {
	return m.TouchLastSeenFn(ctx, id, userAgent, appVersion)
}
func (m *DeviceRepo) GetByDeviceUID(ctx context.Context, deviceUID string) (*model.Device, error) {
	return m.GetByDeviceUIDFn(ctx, deviceUID)
}
func (m *DeviceRepo) FindSingleByUserAgent(ctx context.Context, userID uuid.UUID, deviceType, userAgent string) (*model.Device, error) {
	return m.FindSingleByUserAgentFn(ctx, userID, deviceType, userAgent)
}
func (m *DeviceRepo) SetPrimary(ctx context.Context, userID, deviceID uuid.UUID) error {
	return m.SetPrimaryFn(ctx, userID, deviceID)
}
func (m *DeviceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}

var _ repository.DeviceRepo = (*DeviceRepo)(nil)

// ---------------------------------------------------------------------------
// MapConfigRepo
// ---------------------------------------------------------------------------

// MapConfigRepo is a mock implementation of repository.MapConfigRepo.
type MapConfigRepo struct {
	CreateFn       func(ctx context.Context, mc *model.MapConfig) error
	GetByIDFn      func(ctx context.Context, id uuid.UUID) (*model.MapConfig, error)
	GetDefaultFn   func(ctx context.Context) (*model.MapConfig, error)
	ListFn         func(ctx context.Context) ([]model.MapConfig, error)
	UpdateFn       func(ctx context.Context, mc *model.MapConfig) error
	DeleteFn       func(ctx context.Context, id uuid.UUID) error
	ClearDefaultFn func(ctx context.Context) error
	CountBuiltinFn func(ctx context.Context) (int64, error)
}

func (m *MapConfigRepo) Create(ctx context.Context, mc *model.MapConfig) error {
	return m.CreateFn(ctx, mc)
}
func (m *MapConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.MapConfig, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MapConfigRepo) GetDefault(ctx context.Context) (*model.MapConfig, error) {
	return m.GetDefaultFn(ctx)
}
func (m *MapConfigRepo) List(ctx context.Context) ([]model.MapConfig, error) {
	return m.ListFn(ctx)
}
func (m *MapConfigRepo) Update(ctx context.Context, mc *model.MapConfig) error {
	return m.UpdateFn(ctx, mc)
}
func (m *MapConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *MapConfigRepo) ClearDefault(ctx context.Context) error {
	return m.ClearDefaultFn(ctx)
}
func (m *MapConfigRepo) CountBuiltin(ctx context.Context) (int64, error) {
	return m.CountBuiltinFn(ctx)
}

var _ repository.MapConfigRepo = (*MapConfigRepo)(nil)

// ---------------------------------------------------------------------------
// TerrainConfigRepo
// ---------------------------------------------------------------------------

// TerrainConfigRepo is a mock implementation of repository.TerrainConfigRepo.
type TerrainConfigRepo struct {
	CreateFn       func(ctx context.Context, tc *model.TerrainConfig) error
	GetByIDFn      func(ctx context.Context, id uuid.UUID) (*model.TerrainConfig, error)
	GetDefaultFn   func(ctx context.Context) (*model.TerrainConfig, error)
	ListFn         func(ctx context.Context) ([]model.TerrainConfig, error)
	UpdateFn       func(ctx context.Context, tc *model.TerrainConfig) error
	DeleteFn       func(ctx context.Context, id uuid.UUID) error
	ClearDefaultFn func(ctx context.Context) error
	CountBuiltinFn func(ctx context.Context) (int64, error)
}

func (m *TerrainConfigRepo) Create(ctx context.Context, tc *model.TerrainConfig) error {
	return m.CreateFn(ctx, tc)
}
func (m *TerrainConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.TerrainConfig, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *TerrainConfigRepo) GetDefault(ctx context.Context) (*model.TerrainConfig, error) {
	return m.GetDefaultFn(ctx)
}
func (m *TerrainConfigRepo) List(ctx context.Context) ([]model.TerrainConfig, error) {
	return m.ListFn(ctx)
}
func (m *TerrainConfigRepo) Update(ctx context.Context, tc *model.TerrainConfig) error {
	return m.UpdateFn(ctx, tc)
}
func (m *TerrainConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *TerrainConfigRepo) ClearDefault(ctx context.Context) error {
	return m.ClearDefaultFn(ctx)
}
func (m *TerrainConfigRepo) CountBuiltin(ctx context.Context) (int64, error) {
	return m.CountBuiltinFn(ctx)
}

var _ repository.TerrainConfigRepo = (*TerrainConfigRepo)(nil)

// ---------------------------------------------------------------------------
// ServerSettingsRepo
// ---------------------------------------------------------------------------

// ServerSettingsRepo is a mock implementation of repository.ServerSettingsRepo.
type ServerSettingsRepo struct {
	GetFn    func(ctx context.Context, key string) (*model.ServerSetting, error)
	SetFn    func(ctx context.Context, key, value string) error
	GetAllFn func(ctx context.Context) ([]model.ServerSetting, error)
}

func (m *ServerSettingsRepo) Get(ctx context.Context, key string) (*model.ServerSetting, error) {
	return m.GetFn(ctx, key)
}
func (m *ServerSettingsRepo) Set(ctx context.Context, key, value string) error {
	return m.SetFn(ctx, key, value)
}
func (m *ServerSettingsRepo) GetAll(ctx context.Context) ([]model.ServerSetting, error) {
	return m.GetAllFn(ctx)
}

var _ repository.ServerSettingsRepo = (*ServerSettingsRepo)(nil)

// ---------------------------------------------------------------------------
// MFARepo
// ---------------------------------------------------------------------------

// MFARepo is a mock implementation of repository.MFARepo.
type MFARepo struct {
	// TOTP
	CreateTOTPFn           func(ctx context.Context, m *model.TOTPMethod) error
	GetTOTPByIDFn          func(ctx context.Context, id uuid.UUID) (*model.TOTPMethod, error)
	ListTOTPByUserFn       func(ctx context.Context, userID uuid.UUID) ([]model.TOTPMethod, error)
	VerifyTOTPFn           func(ctx context.Context, id uuid.UUID) error
	TouchTOTPFn            func(ctx context.Context, id uuid.UUID) error
	DeleteTOTPFn           func(ctx context.Context, id uuid.UUID) error
	DeleteAllTOTPForUserFn func(ctx context.Context, userID uuid.UUID) error

	// WebAuthn
	CreateWebAuthnFn              func(ctx context.Context, c *model.WebAuthnCredential) error
	GetWebAuthnByIDFn             func(ctx context.Context, id uuid.UUID) (*model.WebAuthnCredential, error)
	GetWebAuthnByCredentialIDFn   func(ctx context.Context, credentialID []byte) (*model.WebAuthnCredential, error)
	ListWebAuthnByUserFn          func(ctx context.Context, userID uuid.UUID) ([]model.WebAuthnCredential, error)
	ListPasswordlessCredentialsFn func(ctx context.Context) ([]model.WebAuthnCredential, error)
	UpdateWebAuthnSignCountFn     func(ctx context.Context, id uuid.UUID, signCount int64, backupState bool) error
	UpdateWebAuthnPasswordlessFn  func(ctx context.Context, id uuid.UUID, enabled bool) error
	DeleteWebAuthnFn              func(ctx context.Context, id uuid.UUID) error
	DeleteAllWebAuthnForUserFn    func(ctx context.Context, userID uuid.UUID) error

	// Recovery codes
	CreateRecoveryCodesFn           func(ctx context.Context, userID uuid.UUID, hashes []string) error
	ListUnusedRecoveryCodesFn       func(ctx context.Context, userID uuid.UUID) ([]model.RecoveryCode, error)
	MarkRecoveryCodeUsedFn          func(ctx context.Context, id uuid.UUID) error
	DeleteAllRecoveryCodesForUserFn func(ctx context.Context, userID uuid.UUID) error
	CountUnusedRecoveryCodesFn      func(ctx context.Context, userID uuid.UUID) (int, error)

	// Aggregate helpers
	CountVerifiedMethodsFn func(ctx context.Context, userID uuid.UUID) (int, error)
	HasVerifiedTOTPFn      func(ctx context.Context, userID uuid.UUID) (bool, error)
	HasWebAuthnFn          func(ctx context.Context, userID uuid.UUID) (bool, error)
}

func (m *MFARepo) CreateTOTP(ctx context.Context, t *model.TOTPMethod) error {
	return m.CreateTOTPFn(ctx, t)
}
func (m *MFARepo) GetTOTPByID(ctx context.Context, id uuid.UUID) (*model.TOTPMethod, error) {
	return m.GetTOTPByIDFn(ctx, id)
}
func (m *MFARepo) ListTOTPByUser(ctx context.Context, userID uuid.UUID) ([]model.TOTPMethod, error) {
	return m.ListTOTPByUserFn(ctx, userID)
}
func (m *MFARepo) VerifyTOTP(ctx context.Context, id uuid.UUID) error {
	return m.VerifyTOTPFn(ctx, id)
}
func (m *MFARepo) TouchTOTP(ctx context.Context, id uuid.UUID) error {
	return m.TouchTOTPFn(ctx, id)
}
func (m *MFARepo) DeleteTOTP(ctx context.Context, id uuid.UUID) error {
	return m.DeleteTOTPFn(ctx, id)
}
func (m *MFARepo) DeleteAllTOTPForUser(ctx context.Context, userID uuid.UUID) error {
	return m.DeleteAllTOTPForUserFn(ctx, userID)
}
func (m *MFARepo) CreateWebAuthn(ctx context.Context, c *model.WebAuthnCredential) error {
	return m.CreateWebAuthnFn(ctx, c)
}
func (m *MFARepo) GetWebAuthnByID(ctx context.Context, id uuid.UUID) (*model.WebAuthnCredential, error) {
	return m.GetWebAuthnByIDFn(ctx, id)
}
func (m *MFARepo) GetWebAuthnByCredentialID(ctx context.Context, credentialID []byte) (*model.WebAuthnCredential, error) {
	return m.GetWebAuthnByCredentialIDFn(ctx, credentialID)
}
func (m *MFARepo) ListWebAuthnByUser(ctx context.Context, userID uuid.UUID) ([]model.WebAuthnCredential, error) {
	return m.ListWebAuthnByUserFn(ctx, userID)
}
func (m *MFARepo) ListPasswordlessCredentials(ctx context.Context) ([]model.WebAuthnCredential, error) {
	return m.ListPasswordlessCredentialsFn(ctx)
}
func (m *MFARepo) UpdateWebAuthnSignCount(ctx context.Context, id uuid.UUID, signCount int64, backupState bool) error {
	return m.UpdateWebAuthnSignCountFn(ctx, id, signCount, backupState)
}
func (m *MFARepo) UpdateWebAuthnPasswordless(ctx context.Context, id uuid.UUID, enabled bool) error {
	return m.UpdateWebAuthnPasswordlessFn(ctx, id, enabled)
}
func (m *MFARepo) DeleteWebAuthn(ctx context.Context, id uuid.UUID) error {
	return m.DeleteWebAuthnFn(ctx, id)
}
func (m *MFARepo) DeleteAllWebAuthnForUser(ctx context.Context, userID uuid.UUID) error {
	return m.DeleteAllWebAuthnForUserFn(ctx, userID)
}
func (m *MFARepo) CreateRecoveryCodes(ctx context.Context, userID uuid.UUID, hashes []string) error {
	return m.CreateRecoveryCodesFn(ctx, userID, hashes)
}
func (m *MFARepo) ListUnusedRecoveryCodes(ctx context.Context, userID uuid.UUID) ([]model.RecoveryCode, error) {
	return m.ListUnusedRecoveryCodesFn(ctx, userID)
}
func (m *MFARepo) MarkRecoveryCodeUsed(ctx context.Context, id uuid.UUID) error {
	return m.MarkRecoveryCodeUsedFn(ctx, id)
}
func (m *MFARepo) DeleteAllRecoveryCodesForUser(ctx context.Context, userID uuid.UUID) error {
	return m.DeleteAllRecoveryCodesForUserFn(ctx, userID)
}
func (m *MFARepo) CountUnusedRecoveryCodes(ctx context.Context, userID uuid.UUID) (int, error) {
	return m.CountUnusedRecoveryCodesFn(ctx, userID)
}
func (m *MFARepo) CountVerifiedMethods(ctx context.Context, userID uuid.UUID) (int, error) {
	return m.CountVerifiedMethodsFn(ctx, userID)
}
func (m *MFARepo) HasVerifiedTOTP(ctx context.Context, userID uuid.UUID) (bool, error) {
	return m.HasVerifiedTOTPFn(ctx, userID)
}
func (m *MFARepo) HasWebAuthn(ctx context.Context, userID uuid.UUID) (bool, error) {
	return m.HasWebAuthnFn(ctx, userID)
}

var _ repository.MFARepo = (*MFARepo)(nil)

// ---------------------------------------------------------------------------
// APITokenRepo
// ---------------------------------------------------------------------------

// APITokenRepo is a mock implementation of repository.APITokenRepo.
type APITokenRepo struct {
	CreateFn         func(ctx context.Context, token *model.APIToken) error
	GetByTokenHashFn func(ctx context.Context, hash string) (*model.APIToken, *model.User, error)
	ListByUserIDFn   func(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error)
	DeleteFn         func(ctx context.Context, userID, tokenID uuid.UUID) error
	TouchLastUsedFn  func(ctx context.Context, id uuid.UUID) error
	DeleteExpiredFn  func(ctx context.Context) (int64, error)
}

func (m *APITokenRepo) Create(ctx context.Context, token *model.APIToken) error {
	return m.CreateFn(ctx, token)
}
func (m *APITokenRepo) GetByTokenHash(ctx context.Context, hash string) (*model.APIToken, *model.User, error) {
	return m.GetByTokenHashFn(ctx, hash)
}
func (m *APITokenRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	return m.ListByUserIDFn(ctx, userID)
}
func (m *APITokenRepo) Delete(ctx context.Context, userID, tokenID uuid.UUID) error {
	return m.DeleteFn(ctx, userID, tokenID)
}
func (m *APITokenRepo) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	return m.TouchLastUsedFn(ctx, id)
}
func (m *APITokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	return m.DeleteExpiredFn(ctx)
}

var _ repository.APITokenRepo = (*APITokenRepo)(nil)
