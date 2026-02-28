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
];
