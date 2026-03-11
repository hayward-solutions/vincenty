import { http, HttpResponse } from "msw";
import {
  mockAuthResponse,
  mockUser,
  mockAdminUser,
  mockUserList,
  mockGroup,
  mockGroupList,
  mockGroupMember,
  mockDevice,
  mockDeviceResolveMatched,
  mockMessage,
  mockDMPartner,
  mockDrawing,
  mockDrawingShare,
  mockMapSettings,
  mockMapConfig,
  mockTerrainConfig,
  mockAuditLogList,
  mockMFAMethod,
  mockTOTPSetup,
  mockTOTPVerify,
  mockRecoveryCodes,
  mockServerSettings,
  mockLocationHistory,
  mockLatestLocation,
  mockApiToken,
  mockCreateApiTokenResponse,
  mockPermissionPolicy,
} from "./fixtures";

// All handlers use relative URLs (no base URL) since the API client
// uses NEXT_PUBLIC_API_URL which defaults to "".

export const handlers = [
  // -----------------------------------------------------------------------
  // Auth
  // -----------------------------------------------------------------------
  http.post("/api/v1/auth/login", () => {
    return HttpResponse.json(mockAuthResponse);
  }),

  http.post("/api/v1/auth/refresh", () => {
    return HttpResponse.json({
      access_token: "refreshed-access-token",
      refresh_token: "refreshed-refresh-token",
    });
  }),

  http.post("/api/v1/auth/logout", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // -----------------------------------------------------------------------
  // Users
  // -----------------------------------------------------------------------
  http.get("/api/v1/users/me", () => {
    return HttpResponse.json(mockUser);
  }),

  http.put("/api/v1/users/me", () => {
    return HttpResponse.json({ ...mockUser, display_name: "Updated Name" });
  }),

  http.put("/api/v1/users/me/password", () => {
    return HttpResponse.json({ message: "password updated" });
  }),

  http.put("/api/v1/users/me/avatar", () => {
    return HttpResponse.json({ ...mockUser, avatar_url: "/avatars/test.jpg" });
  }),

  http.delete("/api/v1/users/me/avatar", () => {
    return HttpResponse.json({ ...mockUser, avatar_url: "" });
  }),

  http.get("/api/v1/users", () => {
    return HttpResponse.json(mockUserList);
  }),

  http.post("/api/v1/users", () => {
    return HttpResponse.json({ ...mockUser, id: "user-new" });
  }),

  http.put("/api/v1/users/:id", () => {
    return HttpResponse.json(mockUser);
  }),

  http.delete("/api/v1/users/:id", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // -----------------------------------------------------------------------
  // Groups
  // -----------------------------------------------------------------------
  http.get("/api/v1/groups", () => {
    return HttpResponse.json(mockGroupList);
  }),

  http.get("/api/v1/groups/:id", () => {
    return HttpResponse.json(mockGroup);
  }),

  http.post("/api/v1/groups", () => {
    return HttpResponse.json(mockGroup);
  }),

  http.put("/api/v1/groups/:id", () => {
    return HttpResponse.json({ ...mockGroup, name: "Updated Group" });
  }),

  http.delete("/api/v1/groups/:id", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.put("/api/v1/groups/:id/marker", () => {
    return HttpResponse.json(mockGroup);
  }),

  // Group members
  http.get("/api/v1/groups/:groupId/members", () => {
    return HttpResponse.json([mockGroupMember]);
  }),

  http.post("/api/v1/groups/:groupId/members", () => {
    return HttpResponse.json(mockGroupMember);
  }),

  http.put("/api/v1/groups/:groupId/members/:userId", () => {
    return HttpResponse.json({ ...mockGroupMember, is_group_admin: true });
  }),

  http.delete("/api/v1/groups/:groupId/members/:userId", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // -----------------------------------------------------------------------
  // Devices
  // -----------------------------------------------------------------------
  http.get("/api/v1/users/me/devices", () => {
    return HttpResponse.json([mockDevice]);
  }),

  http.post("/api/v1/users/me/devices/resolve", () => {
    return HttpResponse.json(mockDeviceResolveMatched);
  }),

  http.post("/api/v1/users/me/devices/:id/claim", () => {
    return HttpResponse.json(mockDevice);
  }),

  http.post("/api/v1/users/me/devices", () => {
    return HttpResponse.json(mockDevice);
  }),

  http.put("/api/v1/devices/:id", () => {
    return HttpResponse.json({ ...mockDevice, name: "Renamed" });
  }),

  http.put("/api/v1/users/me/devices/:id/primary", () => {
    return HttpResponse.json({ ...mockDevice, is_primary: true });
  }),

  http.delete("/api/v1/devices/:id", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // -----------------------------------------------------------------------
  // Messages
  // -----------------------------------------------------------------------
  http.post("/api/v1/messages", () => {
    return HttpResponse.json(mockMessage);
  }),

  http.get("/api/v1/groups/:groupId/messages", () => {
    return HttpResponse.json([mockMessage]);
  }),

  http.get("/api/v1/messages/direct/:userId", () => {
    return HttpResponse.json([{ ...mockMessage, group_id: undefined, recipient_id: "user-2" }]);
  }),

  http.delete("/api/v1/messages/:id", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.get("/api/v1/messages/conversations", () => {
    return HttpResponse.json([mockDMPartner]);
  }),

  // -----------------------------------------------------------------------
  // User groups (for conversations)
  // -----------------------------------------------------------------------
  http.get("/api/v1/users/me/groups", () => {
    return HttpResponse.json([mockGroup]);
  }),

  // -----------------------------------------------------------------------
  // Drawings
  // -----------------------------------------------------------------------
  http.get("/api/v1/drawings", () => {
    return HttpResponse.json([mockDrawing]);
  }),

  http.get("/api/v1/drawings/shared", () => {
    return HttpResponse.json([{ ...mockDrawing, id: "drawing-shared", owner_id: "user-2" }]);
  }),

  http.post("/api/v1/drawings", () => {
    return HttpResponse.json(mockDrawing);
  }),

  http.put("/api/v1/drawings/:id", () => {
    return HttpResponse.json({ ...mockDrawing, name: "Updated Drawing" });
  }),

  http.delete("/api/v1/drawings/:id", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.post("/api/v1/drawings/:id/share", () => {
    return HttpResponse.json(mockMessage);
  }),

  http.get("/api/v1/drawings/:id/shares", () => {
    return HttpResponse.json([mockDrawingShare]);
  }),

  http.delete("/api/v1/drawings/:drawingId/shares/:messageId", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // -----------------------------------------------------------------------
  // Map settings
  // -----------------------------------------------------------------------
  http.get("/api/v1/map/settings", () => {
    return HttpResponse.json(mockMapSettings);
  }),

  http.get("/api/v1/map-configs", () => {
    return HttpResponse.json([mockMapConfig]);
  }),

  http.post("/api/v1/map-configs", () => {
    return HttpResponse.json(mockMapConfig);
  }),

  http.put("/api/v1/map-configs/:id", () => {
    return HttpResponse.json({ ...mockMapConfig, name: "Updated Map" });
  }),

  http.delete("/api/v1/map-configs/:id", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.get("/api/v1/terrain-configs", () => {
    return HttpResponse.json([mockTerrainConfig]);
  }),

  http.post("/api/v1/terrain-configs", () => {
    return HttpResponse.json(mockTerrainConfig);
  }),

  http.put("/api/v1/terrain-configs/:id", () => {
    return HttpResponse.json({ ...mockTerrainConfig, name: "Updated Terrain" });
  }),

  http.delete("/api/v1/terrain-configs/:id", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // -----------------------------------------------------------------------
  // Audit logs
  // -----------------------------------------------------------------------
  http.get("/api/v1/audit-logs/me", () => {
    return HttpResponse.json(mockAuditLogList);
  }),

  http.get("/api/v1/groups/:groupId/audit-logs", () => {
    return HttpResponse.json(mockAuditLogList);
  }),

  http.get("/api/v1/audit-logs", () => {
    return HttpResponse.json(mockAuditLogList);
  }),

  http.get("/api/v1/audit-logs/me/export", () => {
    return new HttpResponse("id,action\n1,login", {
      headers: { "Content-Type": "text/csv" },
    });
  }),

  http.get("/api/v1/audit-logs/export", () => {
    return new HttpResponse("id,action\n1,login", {
      headers: { "Content-Type": "text/csv" },
    });
  }),

  http.get("/api/v1/users/me/locations/export", () => {
    return new HttpResponse("<gpx></gpx>", {
      headers: { "Content-Type": "application/gpx+xml" },
    });
  }),

  // -----------------------------------------------------------------------
  // MFA
  // -----------------------------------------------------------------------
  http.get("/api/v1/users/me/mfa/methods", () => {
    return HttpResponse.json([mockMFAMethod]);
  }),

  http.post("/api/v1/users/me/mfa/totp/setup", () => {
    return HttpResponse.json(mockTOTPSetup);
  }),

  http.post("/api/v1/users/me/mfa/totp/verify", () => {
    return HttpResponse.json(mockTOTPVerify);
  }),

  http.post("/api/v1/users/me/mfa/webauthn/register/begin", () => {
    return HttpResponse.json({ publicKey: {} });
  }),

  http.post("/api/v1/users/me/mfa/webauthn/register/finish", () => {
    return HttpResponse.json({ registered: true });
  }),

  http.delete("/api/v1/users/me/mfa/methods/:methodId", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.put("/api/v1/users/me/mfa/webauthn/:credId/passwordless", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.post("/api/v1/users/me/mfa/recovery-codes", () => {
    return HttpResponse.json(mockRecoveryCodes);
  }),

  http.post("/api/v1/auth/mfa/totp", () => {
    return HttpResponse.json(mockAuthResponse);
  }),

  http.post("/api/v1/auth/mfa/recovery", () => {
    return HttpResponse.json(mockAuthResponse);
  }),

  http.post("/api/v1/auth/mfa/webauthn/begin", () => {
    return HttpResponse.json({ options: {}, mfa_token: "mfa-token-123" });
  }),

  http.post("/api/v1/auth/mfa/webauthn/finish", () => {
    return HttpResponse.json(mockAuthResponse);
  }),

  http.post("/api/v1/auth/passkey/begin", () => {
    return HttpResponse.json({
      options: { publicKey: { challenge: "test", allowCredentials: [] } },
      session_id: "session-1",
    });
  }),

  http.post("/api/v1/auth/passkey/finish", () => {
    return HttpResponse.json(mockAuthResponse);
  }),

  // Server settings
  http.get("/api/v1/server/settings", () => {
    return HttpResponse.json(mockServerSettings);
  }),

  http.put("/api/v1/server/settings", () => {
    return HttpResponse.json({ ...mockServerSettings, mfa_required: true });
  }),

  // Admin reset MFA
  http.delete("/api/v1/users/:userId/mfa", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // -----------------------------------------------------------------------
  // Locations
  // -----------------------------------------------------------------------
  http.get("/api/v1/groups/:groupId/locations/history", () => {
    return HttpResponse.json([mockLocationHistory]);
  }),

  http.get("/api/v1/users/me/locations/history", () => {
    return HttpResponse.json([mockLocationHistory]);
  }),

  http.get("/api/v1/locations/history", () => {
    return HttpResponse.json([mockLocationHistory]);
  }),

  http.get("/api/v1/users/:userId/locations/history", () => {
    return HttpResponse.json([mockLocationHistory]);
  }),

  http.get("/api/v1/locations", () => {
    return HttpResponse.json([mockLatestLocation]);
  }),

  // -----------------------------------------------------------------------
  // API Tokens
  // -----------------------------------------------------------------------
  http.get("/api/v1/users/me/api-tokens", () => {
    return HttpResponse.json([mockApiToken]);
  }),

  http.post("/api/v1/users/me/api-tokens", () => {
    return HttpResponse.json(mockCreateApiTokenResponse);
  }),

  http.delete("/api/v1/users/me/api-tokens/:id", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // API info (used by About page)
  http.get("/api/v1", () => {
    return HttpResponse.json({ service: "sitaware-api", version: "dev" });
  }),

  // -----------------------------------------------------------------------
  // Permission Policy
  // -----------------------------------------------------------------------
  http.get("/api/v1/server/permissions", () => {
    return HttpResponse.json(mockPermissionPolicy);
  }),

  http.put("/api/v1/server/permissions", async ({ request }) => {
    const body = await request.json();
    return HttpResponse.json(body);
  }),

  // -----------------------------------------------------------------------
  // Media / Calls
  // -----------------------------------------------------------------------
  http.get("/api/v1/calls", () => {
    return HttpResponse.json([]);
  }),

  http.get("/api/v1/groups/:groupId/calls", () => {
    return HttpResponse.json([]);
  }),

  http.post("/api/v1/calls", () => {
    return HttpResponse.json({
      room: { id: "room-1", name: "Test Call", room_type: "call", group_id: null, created_by: "user-1", livekit_room: "lk-room-1", is_active: true, max_participants: 50, created_at: "2025-01-01T00:00:00Z", ended_at: null },
      token: "test-token",
      url: "ws://localhost:7880",
    });
  }),

  http.post("/api/v1/calls/:id/join", () => {
    return HttpResponse.json({
      room: { id: "room-1", name: "Test Call", room_type: "call", group_id: null, created_by: "user-1", livekit_room: "lk-room-1", is_active: true, max_participants: 50, created_at: "2025-01-01T00:00:00Z", ended_at: null },
      token: "test-token",
      url: "ws://localhost:7880",
    });
  }),

  http.post("/api/v1/calls/:id/view", () => {
    return HttpResponse.json({
      room: { id: "room-1", name: "Test Call", room_type: "call", group_id: null, created_by: "user-1", livekit_room: "lk-room-1", is_active: true, max_participants: 50, created_at: "2025-01-01T00:00:00Z", ended_at: null },
      token: "test-token",
      url: "ws://localhost:7880",
    });
  }),

  http.post("/api/v1/calls/:id/leave", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.delete("/api/v1/calls/:id", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // -----------------------------------------------------------------------
  // Video Feeds
  // -----------------------------------------------------------------------
  http.get("/api/v1/groups/:groupId/feeds", () => {
    return HttpResponse.json([]);
  }),

  http.post("/api/v1/feeds", () => {
    return HttpResponse.json({ id: "feed-1", name: "Test Feed", feed_type: "rtmp", group_id: "group-1", created_by: "user-1", is_active: false, created_at: "2025-01-01T00:00:00Z", updated_at: "2025-01-01T00:00:00Z" });
  }),

  http.post("/api/v1/feeds/:id/start", () => {
    return HttpResponse.json({ feed: { id: "feed-1", name: "Test Feed", feed_type: "rtmp", group_id: "group-1", created_by: "user-1", is_active: true, created_at: "2025-01-01T00:00:00Z", updated_at: "2025-01-01T00:00:00Z" }, ingest_url: "rtmp://localhost/live", stream_key: "test-key" });
  }),

  http.post("/api/v1/feeds/:id/stop", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.get("/api/v1/feeds/:id/view", () => {
    return HttpResponse.json({ room: { id: "room-1", name: "Feed Room", room_type: "video_feed" }, token: "test-token", url: "ws://localhost:7880" });
  }),

  http.delete("/api/v1/feeds/:id", () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // -----------------------------------------------------------------------
  // Recordings
  // -----------------------------------------------------------------------
  http.get("/api/v1/calls/:roomId/recordings", () => {
    return HttpResponse.json([]);
  }),

  http.post("/api/v1/recordings/:roomId/start", () => {
    return HttpResponse.json({ id: "rec-1", room_id: "room-1", file_type: "mp4", status: "recording", started_at: "2025-01-01T00:00:00Z" });
  }),

  http.post("/api/v1/recordings/:id/stop", () => {
    return HttpResponse.json({ id: "rec-1", room_id: "room-1", file_type: "mp4", status: "complete", started_at: "2025-01-01T00:00:00Z", ended_at: "2025-01-01T01:00:00Z" });
  }),

  // -----------------------------------------------------------------------
  // PTT Channels
  // -----------------------------------------------------------------------
  http.get("/api/v1/groups/:groupId/ptt-channels", () => {
    return HttpResponse.json([]);
  }),

  http.post("/api/v1/groups/:groupId/ptt-channels", () => {
    return HttpResponse.json({ id: "ptt-1", group_id: "group-1", room_id: "room-ptt-1", name: "Default", is_default: true, created_at: "2025-01-01T00:00:00Z" });
  }),

  http.post("/api/v1/groups/:groupId/ptt-channels/:channelId/join", () => {
    return HttpResponse.json({ channel: { id: "ptt-1", group_id: "group-1", room_id: "room-ptt-1", name: "Default", is_default: true, created_at: "2025-01-01T00:00:00Z" }, token: "ptt-token", url: "ws://localhost:7880" });
  }),

  http.delete("/api/v1/groups/:groupId/ptt-channels/:channelId", () => {
    return new HttpResponse(null, { status: 204 });
  }),
];
