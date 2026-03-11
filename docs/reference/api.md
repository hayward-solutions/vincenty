# API Endpoints

All API routes are prefixed with `/api/v1/`.

## Public

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | Liveness check |
| GET | `/readyz` | Readiness check (DB ping) |
| POST | `/api/v1/auth/login` | Login (returns JWT + refresh token) |
| POST | `/api/v1/auth/refresh` | Rotate tokens |
| POST | `/api/v1/auth/mfa/totp` | Verify TOTP code (MFA challenge) |
| POST | `/api/v1/auth/mfa/webauthn/begin` | Begin WebAuthn assertion (MFA challenge) |
| POST | `/api/v1/auth/mfa/webauthn/finish` | Finish WebAuthn assertion (MFA challenge) |
| POST | `/api/v1/auth/mfa/recovery` | Verify recovery code (MFA challenge) |
| POST | `/api/v1/auth/passkey/begin` | Begin passkey login (passwordless) |
| POST | `/api/v1/auth/passkey/finish` | Finish passkey login (passwordless) |

## Authenticated

| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/auth/logout` | Revoke refresh token |
| GET | `/api/v1/users/me` | Current user profile |
| PUT | `/api/v1/users/me` | Update own profile |
| GET | `/api/v1/users/me/devices` | List own devices |
| POST | `/api/v1/users/me/devices` | Register a device |
| PUT | `/api/v1/devices/{id}` | Update a device |
| DELETE | `/api/v1/devices/{id}` | Delete a device |
| GET | `/api/v1/users/me/groups` | List groups I belong to |
| GET | `/api/v1/groups/{id}/members` | List group members |
| POST | `/api/v1/groups/{id}/members` | Add member to group |
| PUT | `/api/v1/groups/{id}/members/{userId}` | Update member permissions |
| DELETE | `/api/v1/groups/{id}/members/{userId}` | Remove member |
| GET | `/api/v1/groups/{id}/locations/history` | Group location history |
| GET | `/api/v1/map/settings` | Map configuration |
| POST | `/api/v1/messages` | Send message (group or direct) |
| GET | `/api/v1/groups/{id}/messages` | Group message history |
| GET | `/api/v1/messages/conversations` | List DM conversations |
| GET | `/api/v1/messages/direct/{userId}` | Direct message history |
| GET | `/api/v1/messages/{id}` | Get single message |
| DELETE | `/api/v1/messages/{id}` | Delete a message |
| GET | `/api/v1/attachments/{id}/download` | Download file attachment |
| GET | `/api/v1/audit-logs/me` | Own audit logs |
| GET | `/api/v1/audit-logs/me/export` | Export own audit logs |
| GET | `/api/v1/groups/{id}/audit-logs` | Group audit logs |
| GET | `/api/v1/users/me/locations/history` | Own location history |
| GET | `/api/v1/users/me/locations/export` | Export location history (GPX) |
| GET | `/api/v1/ws` | WebSocket connection |
| GET | `/api/v1/mfa/status` | Get own MFA status and methods |
| POST | `/api/v1/mfa/totp/setup` | Begin TOTP setup (returns QR code) |
| POST | `/api/v1/mfa/totp/verify` | Verify TOTP code to activate |
| DELETE | `/api/v1/mfa/totp` | Remove TOTP method |
| POST | `/api/v1/mfa/webauthn/register/begin` | Begin WebAuthn credential registration |
| POST | `/api/v1/mfa/webauthn/register/finish` | Finish WebAuthn credential registration |
| DELETE | `/api/v1/mfa/webauthn/{id}` | Remove a WebAuthn credential |
| PATCH | `/api/v1/mfa/webauthn/{id}` | Update credential (name, passwordless flag) |
| POST | `/api/v1/mfa/recovery/regenerate` | Regenerate recovery codes |
| GET | `/api/v1/users/me/api-tokens` | List own API tokens |
| POST | `/api/v1/users/me/api-tokens` | Create an API token |
| DELETE | `/api/v1/users/me/api-tokens/{id}` | Delete an API token |

## Admin Only

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/users` | List all users |
| POST | `/api/v1/users` | Create user |
| GET | `/api/v1/users/{id}` | Get user |
| PUT | `/api/v1/users/{id}` | Update user |
| DELETE | `/api/v1/users/{id}` | Delete user |
| GET | `/api/v1/groups` | List all groups |
| POST | `/api/v1/groups` | Create group |
| GET | `/api/v1/groups/{id}` | Get group |
| PUT | `/api/v1/groups/{id}` | Update group |
| DELETE | `/api/v1/groups/{id}` | Delete group |
| GET | `/api/v1/locations` | All latest locations |
| GET | `/api/v1/map-configs` | List map configurations |
| POST | `/api/v1/map-configs` | Create map configuration |
| GET | `/api/v1/map-configs/{id}` | Get map configuration |
| PUT | `/api/v1/map-configs/{id}` | Update map configuration |
| DELETE | `/api/v1/map-configs/{id}` | Delete map configuration |
| GET | `/api/v1/audit-logs` | All audit logs |
| GET | `/api/v1/audit-logs/export` | Export all audit logs |
| DELETE | `/api/v1/users/{id}/mfa` | Reset a user's MFA |
| GET | `/api/v1/server-settings` | Get server settings |
| PUT | `/api/v1/server-settings` | Update server settings (e.g. `mfa_required`) |

## WebSocket

Connect via `GET /api/v1/ws?token=<jwt>` or `GET /api/v1/ws?token=sat_<api_token>`.

### Message Types

| Type | Direction | Description |
|---|---|---|
| `location_update` | Client → Server | Send location update |
| `location_update` | Server → Client | Receive group member location |
| `message` | Server → Client | New chat message notification |
| `ping` | Client → Server | Connection liveness check |
| `pong` | Server → Client | Liveness response |
