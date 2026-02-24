import type {
  User,
  AuthResponse,
  Group,
  GroupMember,
  Device,
  DeviceResolveResponse,
  MessageResponse,
  DMConversationPartner,
  DrawingResponse,
  DrawingShareInfo,
  MapSettings,
  MapConfigResponse,
  TerrainConfigResponse,
  AuditLogResponse,
  ListResponse,
  MFAChallengeResponse,
  MFAMethod,
  TOTPSetupResponse,
  TOTPVerifyResponse,
  RecoveryCodesResponse,
  ServerSettings,
  LocationHistoryEntry,
  LatestLocationEntry,
} from "@/types/api";

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

export const mockUser: User = {
  id: "user-1",
  username: "testuser",
  email: "test@example.com",
  display_name: "Test User",
  avatar_url: "",
  marker_icon: "default",
  marker_color: "#3b82f6",
  is_admin: false,
  is_active: true,
  mfa_enabled: false,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const mockAdminUser: User = {
  ...mockUser,
  id: "admin-1",
  username: "admin",
  email: "admin@example.com",
  display_name: "Admin User",
  is_admin: true,
};

export const mockUserList: ListResponse<User> = {
  data: [mockUser, mockAdminUser],
  total: 2,
  page: 1,
  page_size: 20,
};

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export const mockAuthResponse: AuthResponse = {
  access_token: "test-access-token",
  refresh_token: "test-refresh-token",
  user: mockUser,
};

export const mockMFAChallenge: MFAChallengeResponse = {
  mfa_required: true,
  mfa_token: "mfa-token-123",
  methods: ["totp", "recovery"],
};

// ---------------------------------------------------------------------------
// Groups
// ---------------------------------------------------------------------------

export const mockGroup: Group = {
  id: "group-1",
  name: "Test Group",
  description: "A test group",
  marker_icon: "default",
  marker_color: "#10b981",
  created_by: "user-1",
  member_count: 2,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const mockGroupList: ListResponse<Group> = {
  data: [mockGroup],
  total: 1,
  page: 1,
  page_size: 20,
};

export const mockGroupMember: GroupMember = {
  id: "member-1",
  group_id: "group-1",
  user_id: "user-1",
  username: "testuser",
  display_name: "Test User",
  can_read: true,
  can_write: true,
  is_group_admin: false,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

// ---------------------------------------------------------------------------
// Devices
// ---------------------------------------------------------------------------

export const mockDevice: Device = {
  id: "device-1",
  user_id: "user-1",
  name: "Web Browser",
  device_type: "web",
  device_uid: "uid-123",
  is_primary: true,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const mockDeviceResolveMatched: DeviceResolveResponse = {
  matched: true,
  device: mockDevice,
};

export const mockDeviceResolveUnmatched: DeviceResolveResponse = {
  matched: false,
  existing_devices: [mockDevice],
};

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

export const mockMessage: MessageResponse = {
  id: "msg-1",
  sender_id: "user-1",
  username: "testuser",
  display_name: "Test User",
  group_id: "group-1",
  content: "Hello, world!",
  message_type: "text",
  attachments: [],
  created_at: "2025-01-01T00:00:00Z",
};

export const mockDMPartner: DMConversationPartner = {
  user_id: "user-2",
  username: "otheruser",
  display_name: "Other User",
};

// ---------------------------------------------------------------------------
// Drawings
// ---------------------------------------------------------------------------

export const mockDrawing: DrawingResponse = {
  id: "drawing-1",
  owner_id: "user-1",
  username: "testuser",
  display_name: "Test User",
  name: "Test Drawing",
  geojson: { type: "FeatureCollection", features: [] },
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const mockDrawingShare: DrawingShareInfo = {
  type: "group",
  id: "group-1",
  name: "Test Group",
  shared_at: "2025-01-01T00:00:00Z",
  message_id: "msg-share-1",
};

// ---------------------------------------------------------------------------
// Map settings
// ---------------------------------------------------------------------------

export const mockMapConfig: MapConfigResponse = {
  id: "mapconfig-1",
  name: "OpenStreetMap",
  source_type: "raster",
  tile_url: "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
  min_zoom: 0,
  max_zoom: 19,
  is_default: true,
  is_builtin: true,
  is_enabled: true,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const mockTerrainConfig: TerrainConfigResponse = {
  id: "terrain-1",
  name: "Default Terrain",
  source_type: "raster-dem",
  terrain_url: "https://example.com/terrain/{z}/{x}/{y}.png",
  terrain_encoding: "terrarium",
  is_default: true,
  is_builtin: true,
  is_enabled: true,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const mockMapSettings: MapSettings = {
  tile_url: "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
  center_lat: 0,
  center_lng: 0,
  zoom: 2,
  min_zoom: 0,
  max_zoom: 19,
  terrain_url: "",
  terrain_encoding: "terrarium",
  configs: [mockMapConfig],
};

// ---------------------------------------------------------------------------
// Audit logs
// ---------------------------------------------------------------------------

export const mockAuditLog: AuditLogResponse = {
  id: "audit-1",
  user_id: "user-1",
  username: "testuser",
  display_name: "Test User",
  action: "login",
  resource_type: "session",
  ip_address: "127.0.0.1",
  created_at: "2025-01-01T00:00:00Z",
};

export const mockAuditLogList: ListResponse<AuditLogResponse> = {
  data: [mockAuditLog],
  total: 1,
  page: 1,
  page_size: 20,
};

// ---------------------------------------------------------------------------
// MFA
// ---------------------------------------------------------------------------

export const mockMFAMethod: MFAMethod = {
  id: "mfa-method-1",
  type: "totp",
  name: "Authenticator",
  verified: true,
  created_at: "2025-01-01T00:00:00Z",
};

export const mockTOTPSetup: TOTPSetupResponse = {
  method_id: "mfa-method-1",
  secret: "JBSWY3DPEHPK3PXP",
  uri: "otpauth://totp/SitAware:testuser?secret=JBSWY3DPEHPK3PXP&issuer=SitAware",
  issuer: "SitAware",
  account: "testuser",
};

export const mockTOTPVerify: TOTPVerifyResponse = {
  verified: true,
  recovery_codes: ["aaaa-bbbb", "cccc-dddd", "eeee-ffff"],
};

export const mockRecoveryCodes: RecoveryCodesResponse = {
  codes: ["aaaa-bbbb", "cccc-dddd", "eeee-ffff", "gggg-hhhh"],
};

export const mockServerSettings: ServerSettings = {
  mfa_required: false,
  mapbox_access_token: "",
  google_maps_api_key: "",
};

// ---------------------------------------------------------------------------
// Locations
// ---------------------------------------------------------------------------

export const mockLocationHistory: LocationHistoryEntry = {
  user_id: "user-1",
  device_id: "device-1",
  device_name: "Web Browser",
  username: "testuser",
  display_name: "Test User",
  lat: -33.8688,
  lng: 151.2093,
  recorded_at: "2025-01-01T12:00:00Z",
};

export const mockLatestLocation: LatestLocationEntry = {
  user_id: "user-1",
  device_id: "device-1",
  device_name: "Web Browser",
  is_primary: true,
  username: "testuser",
  display_name: "Test User",
  lat: -33.8688,
  lng: 151.2093,
  recorded_at: "2025-01-01T12:00:00Z",
};
