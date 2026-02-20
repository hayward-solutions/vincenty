export interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  avatar_url: string;
  is_admin: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Device {
  id: string;
  user_id: string;
  name: string;
  device_type: string;
  device_uid: string;
  user_agent?: string;
  last_seen_at?: string;
  created_at: string;
  updated_at: string;
}

export interface DeviceResolveResponse {
  matched: boolean;
  device?: Device;
  existing_devices?: Device[];
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

export interface ListResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface ApiErrorResponse {
  error: {
    code: string;
    message: string;
  };
}

export interface CreateUserRequest {
  username: string;
  email: string;
  password: string;
  display_name?: string;
  is_admin?: boolean;
}

export interface UpdateUserRequest {
  email?: string;
  display_name?: string;
  password?: string;
  is_admin?: boolean;
  is_active?: boolean;
}

export interface UpdateMeRequest {
  email?: string;
  display_name?: string;
}

export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
}

export interface Group {
  id: string;
  name: string;
  description: string;
  created_by?: string;
  member_count: number;
  created_at: string;
  updated_at: string;
}

export interface GroupMember {
  id: string;
  group_id: string;
  user_id: string;
  username: string;
  display_name: string;
  can_read: boolean;
  can_write: boolean;
  is_group_admin: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateGroupRequest {
  name: string;
  description?: string;
}

export interface UpdateGroupRequest {
  name?: string;
  description?: string;
}

export interface AddGroupMemberRequest {
  user_id: string;
  can_read?: boolean;
  can_write?: boolean;
  is_group_admin?: boolean;
}

export interface UpdateGroupMemberRequest {
  can_read?: boolean;
  can_write?: boolean;
  is_group_admin?: boolean;
}

// ---------------------------------------------------------------------------
// WebSocket message types
// ---------------------------------------------------------------------------

export interface WSEnvelope {
  type: string;
  payload: unknown;
}

/** Client → Server: send current position */
export interface WSLocationUpdate {
  device_id: string;
  lat: number;
  lng: number;
  altitude?: number;
  heading?: number;
  speed?: number;
  accuracy?: number;
}

/** Server → Client: another user's position update */
export interface WSLocationBroadcast {
  user_id: string;
  username: string;
  display_name: string;
  group_id: string;
  lat: number;
  lng: number;
  altitude?: number;
  heading?: number;
  speed?: number;
  timestamp: string;
}

/** Server → Client: initial snapshot of group member positions */
export interface WSLocationSnapshot {
  group_id: string;
  locations: WSLocationBroadcast[];
}

/** Server → Client: connection acknowledgement */
export interface WSConnected {
  user_id: string;
  groups: { id: string; name: string }[];
}

/** Server → Client: error */
export interface WSError {
  message: string;
}

// ---------------------------------------------------------------------------
// Map configuration
// ---------------------------------------------------------------------------

export interface MapConfigResponse {
  id: string;
  name: string;
  source_type: string;
  tile_url: string;
  style_json?: Record<string, unknown>;
  min_zoom: number;
  max_zoom: number;
  is_default: boolean;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface MapSettings {
  tile_url: string;
  style_json?: Record<string, unknown>;
  center_lat: number;
  center_lng: number;
  zoom: number;
  min_zoom: number;
  max_zoom: number;
  configs: MapConfigResponse[];
}

export interface CreateMapConfigRequest {
  name: string;
  source_type?: string;
  tile_url?: string;
  style_json?: Record<string, unknown>;
  min_zoom?: number;
  max_zoom?: number;
  is_default?: boolean;
}

export interface UpdateMapConfigRequest {
  name?: string;
  source_type?: string;
  tile_url?: string;
  style_json?: Record<string, unknown>;
  min_zoom?: number;
  max_zoom?: number;
  is_default?: boolean;
}

// ---------------------------------------------------------------------------
// Location history
// ---------------------------------------------------------------------------

export interface LocationHistoryEntry {
  user_id: string;
  device_id: string;
  device_name: string;
  username: string;
  display_name: string;
  lat: number;
  lng: number;
  altitude?: number;
  heading?: number;
  speed?: number;
  recorded_at: string;
}

export interface LatestLocationEntry {
  user_id: string;
  device_id: string;
  device_name: string;
  username: string;
  display_name: string;
  lat: number;
  lng: number;
  altitude?: number;
  heading?: number;
  speed?: number;
  recorded_at: string;
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

export interface Attachment {
  id: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
}

export interface MessageResponse {
  id: string;
  sender_id: string;
  username: string;
  display_name: string;
  group_id?: string;
  recipient_id?: string;
  content: string;
  message_type: string;
  lat?: number;
  lng?: number;
  metadata?: unknown;
  attachments: Attachment[];
  created_at: string;
}

/** Server → Client: a new message via WebSocket */
export interface WSMessageNew extends MessageResponse {}

/** A user the caller has DM history with (from GET /api/v1/messages/conversations) */
export interface DMConversationPartner {
  user_id: string;
  username: string;
  display_name: string;
}

/** Conversation list item — either a group or a DM partner */
export interface Conversation {
  id: string; // group_id or user_id
  type: "group" | "direct";
  name: string;
  lastMessage?: MessageResponse;
}

// ---------------------------------------------------------------------------
// Audit logs
// ---------------------------------------------------------------------------

export interface AuditLogResponse {
  id: string;
  user_id: string;
  username: string;
  display_name: string;
  device_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  group_id?: string;
  metadata?: unknown;
  lat?: number;
  lng?: number;
  ip_address: string;
  created_at: string;
}

export interface AuditFilters {
  from?: string;
  to?: string;
  action?: string;
  resource_type?: string;
  page?: number;
  page_size?: number;
}
