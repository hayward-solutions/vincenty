# Architecture

## System Overview

```
┌──────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   Browser    │  │  ATAK/iTAK  │  │   iOS App   │  │     CLI     │
│  (Next.js)   │  │  (CoT XML)  │  │  (SwiftUI)  │  │  (Go, GPX)  │
└──────┬───────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                 │                 │                 │
       │  HTTP/WS        │  HTTP           │  HTTP/WS        │  HTTP/WS
       │                 │                 │                 │
┌──────▼────────────────────▼────────────────────▼──────┐
│              Load Balancer / Reverse Proxy             │
│       (Caddy, ALB, Ingress — handles TLS)             │
└──────────────────────┬────────────────────────────────┘
                       │
┌──────────────────────▼────────────────────────────────┐
│                   Go API Service                      │
│                                                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ Handlers │→ │ Services │→ │  Repos   │            │
│  │  (HTTP)  │  │ (Logic)  │  │  (SQL)   │            │
│  └──────────┘  └──────────┘  └────┬─────┘            │
│       │                           │                   │
│  ┌────▼─────┐              ┌──────▼──────┐            │
│  │WebSocket │              │  Pub/Sub    │            │
│  │   Hub    │◄─────────────│ (Interface) │            │
│  └──────────┘              └─────────────┘            │
└────────┬──────────────────────┬──────────┬────────────┘
         │                      │          │
    ┌────▼─────┐          ┌─────▼────┐  ┌──▼─────┐
    │PostgreSQL│          │  Redis   │  │S3/Minio│
    │+ PostGIS │          │(Pub/Sub) │  │(Files) │
    └──────────┘          └──────────┘  └────────┘
```

The system is composed of three application services (Go API, Next.js web client, iOS client) backed by three infrastructure services (PostgreSQL with PostGIS, Redis, S3-compatible object storage). A reverse proxy (Caddy, ALB, or Ingress controller) sits in front for TLS termination and routing.

## API Service — Layered Architecture

The API follows a strict layered design. Each layer only depends on the one below it. There are no circular dependencies.

```
HTTP Request
     │
     ▼
┌─────────────┐
│  Middleware  │  CORS → MaxBodySize → RateLimit → Logging → Audit → Router
└──────┬──────┘
       ▼
┌─────────────┐
│   Handler   │  Parse request, validate input, call service, write JSON response
└──────┬──────┘
       ▼
┌─────────────┐
│   Service   │  Business logic, authorization, orchestration
└──────┬──────┘
       ▼
┌─────────────┐
│ Repository  │  SQL queries via pgx, maps rows to domain models
└──────┬──────┘
       ▼
   PostgreSQL
```

### Layer Responsibilities

**Middleware** (`internal/middleware/`) — Cross-cutting concerns applied to every request. Each middleware follows the stdlib pattern `func(http.Handler) http.Handler` and is composed in `main.go`. The stack is applied in this order (outermost first):

1. **CORS** — Validates `Origin` header against configured allowed origins, sets response headers
2. **MaxBodySize** — Wraps request body with `http.MaxBytesReader` to enforce size limits
3. **RateLimit** — Per-IP token bucket rate limiting using `golang.org/x/time/rate`
4. **Logging** — Structured JSON request/response logging (method, path, status, duration)
5. **Audit** — Records API actions to the audit log (parses JWT from Authorization header independently)
6. **Auth** — Per-route via `authMW.Authenticate()` or `authMW.RequireAdmin()`, injecting user claims into request context

**Handler** (`internal/handler/`) — One file per domain (auth, users, devices, groups, locations, messages, map configs, audit, CoT). Handlers parse HTTP input, delegate to a service, and write JSON responses. They never contain business logic or SQL.

**Service** (`internal/service/`) — Business rules, authorization checks, and cross-domain orchestration. Services call repositories and may call other services (e.g., location service publishes via pub/sub after writing to the repository).

**Repository** (`internal/repository/`) — Pure data access. Each repository owns queries for a single table or closely related set of tables. Uses `pgx/v5` directly — no ORM.

**Model** (`internal/model/`) — Domain models, request/response DTOs, and a typed error system. Models are plain Go structs with no database or HTTP dependencies.

## Authentication

```
Login:
  Client → POST /api/v1/auth/login { username, password }
  Server → Verify password (bcrypt, cost 12)
         → Generate JWT access token (15min, HMAC-SHA256)
         → Generate opaque refresh token (crypto/rand UUID)
         → Store SHA-256(refresh_token) in DB with expiry
         → Return { access_token, refresh_token, user }

Authenticated Request:
  Client → Authorization: Bearer <access_token>
  Server → Middleware extracts JWT, validates signature + expiry
         → Injects claims { user_id, username, is_admin } into context
         → Handler reads claims from context

Token Refresh:
  Client → POST /api/v1/auth/refresh { refresh_token }
  Server → SHA-256 hash the token, look up in DB
         → Verify not expired
         → Delete old token (rotation — each token is single-use)
         → Issue new access + refresh token pair

Auto-Refresh (Web Client):
  api.ts intercepts 401 responses →
    Calls /auth/refresh with stored token →
    Updates localStorage →
    Retries original request (once)
```

Refresh tokens are never stored in plaintext. Only their SHA-256 hash exists in the database. Token rotation means a stolen refresh token can only be used once before it's invalidated.

## Real-Time Architecture

WebSocket connections are managed by a central Hub that runs as a goroutine. The Hub maintains a registry of connected clients, handles subscription to pub/sub channels, and routes messages.

```
Client A (writer)                      Client B (reader)
     │                                      ▲
     │ WS: location_update                  │ WS: location_update
     ▼                                      │
┌──────────┐                          ┌─────┴──────┐
│ API Node │                          │ API Node   │
│   Hub    │                          │   Hub      │
└────┬─────┘                          └─────▲──────┘
     │                                      │
     │ PUBLISH group:123:location           │ SUBSCRIBE group:123:location
     ▼                                      │
┌───────────────────────────────────────────────┐
│                    Redis                      │
│                 (Pub/Sub broker)               │
└───────────────────────────────────────────────┘
```

### Why Redis Pub/Sub?

Multiple API instances can run behind a load balancer. When Client A sends a location update to Node 1, Redis broadcasts it to all subscribed nodes. Node 2's Hub delivers it to Client B. This enables horizontal scaling without sticky sessions for WebSocket.

### Redis Cluster Mode

The API supports both standalone Redis and Redis Cluster mode. Set `REDIS_CLUSTER=true` when using Amazon ElastiCache with cluster mode enabled (or any Redis Cluster deployment). In cluster mode, the client discovers shard topology automatically via the `CLUSTER SLOTS` command using a single configuration endpoint. The pub/sub and MFA session layers use `redis.UniversalClient` and `redis.Cmdable` interfaces respectively, which work transparently across both modes.

### Pub/Sub Channels

| Channel Pattern | Purpose |
|---|---|
| `group:{id}:location` | Real-time location updates for group members |
| `group:{id}:messages` | New messages in a group |
| `user:{id}:direct` | Direct messages to a specific user |
| `global:admin` | Admin-level notifications |

### WebSocket Message Flow

1. Client connects via `GET /api/v1/ws?token=<jwt>`
2. Handler validates JWT, looks up user's groups and permissions
3. Hub registers client, subscribes to relevant Redis channels
4. Client sends location updates → API writes to DB + publishes to Redis
5. Other clients in the same group receive the update via their Hub subscription

### Graceful Shutdown

On SIGINT/SIGTERM:
1. Hub context is cancelled
2. Hub sends close frames to all connected clients
3. 2-second drain period allows write pumps to flush close frames
4. HTTP server graceful shutdown (10-second timeout) completes in-flight requests

## Database Schema

PostgreSQL 16 with PostGIS for spatial operations. The schema contains 17 tables across core, auth, MFA, messaging, spatial, and configuration domains.

```
users ──────< devices
  │              └── is_primary (unique partial index, one per user)
  │
  ├────────< group_members >──────── groups
  │                                    └── marker_icon, marker_color
  │
  ├────────< messages ────────────── attachments
  │              │
  │              └── (group_id OR recipient_id — CHECK constraint)
  │
  ├────────< location_history
  │
  ├────────< refresh_tokens
  │
  ├────────< audit_logs
  │              └── (optional group_id)
  │
  ├────────< user_totp_methods        (encrypted TOTP secrets)
  │
  ├────────< webauthn_credentials     (FIDO2 / passkey public keys)
  │
  ├────────< recovery_codes           (bcrypt-hashed one-time codes)
  │
  └────────< drawings                 (GeoJSON map annotations)

cot_events         (Cursor on Target — ATAK/iTAK ingestion)
map_configs        (tile source configurations, admin-managed)
terrain_configs    (terrain/elevation sources, admin-managed)
server_settings    (key-value server configuration, e.g. mfa_required)
```

### Table Summary

| Table | Purpose | Key Spatial Fields |
|---|---|---|
| `users` | User accounts, roles, avatar, marker style, MFA flag | — |
| `devices` | Registered devices per user, primary device flag | `last_location` GEOMETRY(POINT, 4326) |
| `groups` | Teams/units with marker customization | — |
| `group_members` | User-group membership with granular permissions | — |
| `messages` | Group and direct messages with sender location | `location` GEOMETRY(POINT, 4326) |
| `attachments` | File attachments stored in S3 | — |
| `location_history` | Every location update for replay/export | `location` GEOMETRY(POINT, 4326) |
| `refresh_tokens` | SHA-256 hashed rotating refresh tokens | — |
| `api_tokens` | Long-lived API tokens for CLI/programmatic access | — |
| `audit_logs` | Automatic API action audit trail | `location` GEOMETRY(POINT, 4326) |
| `cot_events` | Cursor on Target XML events from ATAK/iTAK | `location` GEOMETRY(POINT, 4326) |
| `user_totp_methods` | Encrypted TOTP secrets (AES-256-GCM or KMS) | — |
| `webauthn_credentials` | WebAuthn/FIDO2 public keys and metadata | — |
| `recovery_codes` | One-time bcrypt-hashed recovery codes | — |
| `drawings` | GeoJSON map annotations (lines, circles, rects) | — |
| `map_configs` | Map tile source configurations | — |
| `terrain_configs` | Terrain/elevation source configurations | — |
| `server_settings` | Key-value server settings (e.g. `mfa_required`) | — |

### Spatial Data

Location fields use `GEOMETRY(POINT, 4326)` with GIST spatial indexes. The `location_history` table stores every location update for replay. All coordinates use WGS 84 (SRID 4326).

### Migrations

Migrations are embedded in the Go binary via `//go:embed` and run automatically on startup using `golang-migrate`. Files are in `services/api/internal/database/migrations/` with the naming convention `{number}_{description}.{up|down}.sql`.

| Migration | Description |
|---|---|
| 000001 | Enable PostGIS and uuid-ossp extensions |
| 000002 | Create core tables (users, devices, groups, members, messages, attachments, locations, map_configs, audit_logs) |
| 000003 | Create indexes (spatial GIST, temporal DESC, foreign key — 14 indexes total) |
| 000004 | Create refresh_tokens table with token_hash unique index |
| 000005 | Create cot_events table with 8 indexes (spatial, temporal, UID lookup) |
| 000006 | Add avatar_url column to users |
| 000007 | Add user_agent to devices with partial index for heuristic device recognition |
| 000008 | Add marker_icon and marker_color to groups |
| 000009 | Add marker_icon and marker_color to users |
| 000010 | Add MFA support: user_totp_methods, webauthn_credentials, recovery_codes tables; server_settings table; mfa_enabled flag on users |
| 000011 | Add WebAuthn backup_eligible and backup_state flags (go-webauthn v0.15+ compat) |
| 000012 | Add terrain_url and terrain_encoding to map_configs |
| 000013 | Extract terrain into separate terrain_configs table, migrate data, drop terrain columns from map_configs |
| 000014 | Add source_type to terrain_configs |
| 000015 | Add is_builtin and is_enabled flags to map_configs and terrain_configs |
| 000016 | Create drawings table for GeoJSON map annotations |
| 000017 | Add is_primary flag to devices with unique partial index (one primary per user), backfill oldest device |
| 000018 | Create api_tokens table with user FK, SHA-256 token hash unique index, optional expiry |

## Pub/Sub Interface

The pub/sub layer is abstracted behind a Go interface to allow swapping implementations:

```go
type PubSub interface {
    Publish(ctx context.Context, channel string, msg []byte) error
    Subscribe(ctx context.Context, channels ...string) (Subscription, error)
    Close() error
}

type Subscription interface {
    Channel() <-chan Message
    Unsubscribe() error
}
```

Current implementation: Redis (`internal/pubsub/redis.go`). The interface is designed to support Kafka or Apache Ignite as future backends without changing any business logic.

## Object Storage Interface

File storage is abstracted behind an interface:

```go
type ObjectStore interface {
    Upload(ctx context.Context, key string, r io.Reader, contentType string, size int64) error
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    PresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}
```

Implementation uses AWS SDK v2, which works with both AWS S3 and Minio. In development, Minio runs locally in Docker. In production, use AWS S3 directly (or any S3-compatible service) — no Minio needed.

## Web Client Architecture

### Route Structure

| Route | Auth | Description |
|---|---|---|
| `/login` | Public | Login form (password + passkey) |
| `/dashboard` | Authenticated | Overview, connection status, quick links |
| `/map` | Authenticated | Real-time map with location markers, drawing, replay, measurement |
| `/messages` | Authenticated | Group and direct messaging with file attachments |
| `/settings/account/general` | Authenticated | Profile, avatar upload, map marker customization |
| `/settings/account/security` | Authenticated | Password change, MFA setup (TOTP, WebAuthn, recovery codes) |
| `/settings/account/devices` | Authenticated | Device management, primary device |
| `/settings/account/activity` | Authenticated | Personal audit log |
| `/settings/account/groups` | Authenticated | View own group memberships |
| `/settings/server/map` | Admin | Map tile and terrain source configuration, API keys |
| `/settings/server/users` | Admin | User CRUD, role assignment, MFA status |
| `/settings/server/groups` | Admin | Group CRUD, membership management |
| `/settings/server/groups/[id]` | Admin | Group detail with member list |
| `/settings/server/security` | Admin | Server-wide MFA enforcement policy |
| `/settings/server/audit-logs` | Admin | Full audit log viewer with filters and export |

### Data Flow

```
React Component → Custom Hook (useUsers, useMessages, etc.)
                       │
                       ▼
                  api.ts (ApiClient)
                       │
                       ├── Attaches JWT from localStorage
                       ├── Intercepts 401 → auto-refresh → retry
                       │
                       ▼
                  Next.js Rewrite (/api/* → Go API)
                       │
                       ▼
                  Go API Service
```

### WebSocket Integration

The `WebSocketContext` provider connects on login and disconnects on logout. It provides:
- Real-time location updates (rendered as markers on the map)
- Incoming message notifications
- Location sharing (browser Geolocation API → WebSocket → API → group members)

## iOS Client Architecture

The iOS client (`clients/ios/`) is a native SwiftUI app targeting iOS 17+, built with Swift 6 and strict concurrency. It uses the MVVM pattern with the Observation framework (`@Observable` macro).

### Service Layer

```
SwiftUI View → @Observable ViewModel / @Environment Service
                     │
                     ▼
               APIClient.shared
                     │
                     ├── Attaches JWT from Keychain (via TokenManager)
                     ├── Intercepts 401 → auto-refresh → retry
                     │
                     ▼
               Go API Service
```

Key services injected via SwiftUI `.environment()`:

| Service | Responsibility |
|---|---|
| `APIClient` | Singleton HTTP client. All network calls route through it. Token refresh is actor-isolated with single-flight deduplication |
| `AuthManager` | Auth state (login, logout, passkey login, MFA verification). Publishes `currentUser` and `isAuthenticated` |
| `WebSocketService` | Persistent real-time connection for location updates and messages |
| `LocationSharingManager` | Wraps `CLLocationManager` for background location updates |
| `SyncManager` | Offline mutation queue (SwiftData). Drains FIFO on reconnect with server-wins conflict resolution |
| `DeviceManager` | Device registration and primary device resolution |
| `PasskeyManager` | Singleton wrapping `ASAuthorizationController` for WebAuthn ceremonies (passkey login, MFA, credential registration) |

### Authentication and Passkeys

The iOS client supports three authentication flows:

1. **Password login** — Standard username/password with optional MFA challenge (TOTP, WebAuthn, or recovery code)
2. **Passkey login** — Passwordless via `ASAuthorizationPlatformPublicKeyCredentialProvider`. The app calls `POST /api/v1/auth/passkey/begin` to get a challenge, presents the system Face ID / Touch ID prompt, then sends the assertion to `POST /api/v1/auth/passkey/finish`
3. **WebAuthn MFA** — During an MFA challenge, the app calls the WebAuthn begin/finish endpoints with the MFA token. Uses the same `PasskeyManager` singleton

All WebAuthn binary fields use Base64URL encoding (RFC 4648 §5). A dedicated `WebAuthnJSON.encoder` (no key strategy) avoids the snake_case mangling that `APIClient`’s default encoder applies.

### Offline Support

SwiftData models in `Core/Storage/PersistentModels.swift` cache API data locally. When the device is offline, `SyncManager` queues mutations and drains them FIFO on reconnect. Conflict resolution uses server-wins semantics.

### Map

MapLibre Native iOS SDK wrapped in a `UIViewRepresentable`. Imperative controller classes (`LocationMarkersController`, `DrawToolController`, `MeasureToolController`) manage MapLibre sources and layers directly, keeping the SwiftUI layer declarative.

## Middleware Stack

Applied to every request in this order (outermost first):

```
Request → CORS → MaxBodySize → RateLimit → Logging → Audit → Router → Handler
```

| Middleware | Description |
|---|---|
| **CORS** | Configurable origin validation. Reflects matched origin, sends `Vary: Origin`. Wildcard `*` allowed for dev |
| **MaxBodySize** | Wraps `r.Body` with `http.MaxBytesReader`. Default 10MB. Prevents memory exhaustion |
| **RateLimit** | Per-IP token bucket (`golang.org/x/time/rate`). Returns 429 with `Retry-After`. Stale entries cleaned periodically |
| **Logging** | JSON structured logs: method, path, status code, duration, remote IP |
| **Audit** | Records action, resource, user, IP, location to audit_logs table. Parses JWT independently (no dependency on auth middleware) |

Per-route auth middleware (`Authenticate`, `RequireAdmin`, `AuthenticateWithQueryToken`) is applied individually in the route registration, not globally.

## Deployment Targets

### Development — Docker Compose

`docker-compose.yml` runs the full stack: PostgreSQL+PostGIS, Redis, Minio (with auto-bucket-creation init container), Go API, and Next.js web client. Minio provides S3-compatible storage locally.

### Production — Docker Compose + Caddy

`docker-compose.prod.yml` adds Caddy for TLS termination and removes Minio. No ports are exposed except 80/443 through Caddy. Resource limits are applied to all containers.

### Kubernetes

Raw manifests in `deploy/k8s/` (namespace, configmap, secret, StatefulSet for PostgreSQL, deployments for Redis/API/web, services, Ingress with TLS). API runs 2 replicas behind a Service.

### Helm

Full Helm chart in `deploy/helm/sitaware/` with configurable values for all resources. Supports toggling between in-cluster PostgreSQL/Redis and external managed services (RDS, ElastiCache).

### AWS ECS Fargate

Task definitions and service configs in `deploy/ecs/`. Uses ALB for TLS termination, RDS for PostgreSQL, ElastiCache for Redis, S3 for storage, SSM Parameter Store for secrets, and IAM task roles for S3 access (no static credentials).

## Configuration Philosophy

- **Environment variables only** — no config files, no CLI flags. Every setting has a sensible default
- **No prefix** — variables are named directly (`DB_HOST`, not `SITAWARE_DB_HOST`)
- **Build-time vs runtime** — `NEXT_PUBLIC_*` variables are inlined at Next.js build time. All others are runtime
- **Secret management** — in Docker Compose: `.env` file. In Kubernetes: Secrets. In ECS: SSM Parameter Store
- **Migrations are automatic** — the API runs migrations on startup. No separate migration step needed

## Multi-Factor Authentication

SitAware supports three MFA methods, all optional per-user unless the admin enables server-wide enforcement.

```
┌─────────────────────────────────────────────────────┐
│                   MFA Methods                       │
├──────────────────┬──────────────────┬───────────────┤
│  TOTP            │  WebAuthn/FIDO2  │  Recovery     │
│  (Authenticator  │  (Security Keys, │  Codes        │
│   Apps)          │   Passkeys)      │  (8 one-time) │
├──────────────────┴──────────────────┴───────────────┤
│             Server-wide enforcement                  │
│  server_settings.mfa_required = true/false           │
│  MFA middleware blocks access until MFA is set up    │
└─────────────────────────────────────────────────────┘
```

### TOTP Flow

1. User calls `POST /api/v1/mfa/totp/setup` — server generates a secret, encrypts it (AES-256-GCM via HKDF from `JWT_SECRET`, or AWS KMS), stores in `user_totp_methods`, returns QR code URI
2. User scans QR with authenticator app, submits code to `POST /api/v1/mfa/totp/verify`
3. Server validates the code, marks method as `verified`, enables `mfa_enabled` on user, generates 8 recovery codes (bcrypt-hashed, stored in `recovery_codes`)
4. On login, if MFA is enabled: login returns `mfa_required: true` with a short-lived MFA token. User submits TOTP code to `POST /api/v1/auth/mfa/totp` to complete login

### WebAuthn / Passkey Flow

1. User calls `POST /api/v1/mfa/webauthn/register/begin` — server generates a challenge via `go-webauthn/webauthn`
2. Browser prompts for security key / biometric, returns attestation to `POST /api/v1/mfa/webauthn/register/finish`
3. Server stores the public key in `webauthn_credentials` with transport hints, AAGUID, sign count
4. Passkey login (passwordless): `POST /api/v1/auth/passkey/begin` → browser assertion → `POST /api/v1/auth/passkey/finish`

### TOTP Secret Encryption

TOTP secrets are never stored in plaintext. Two encryption backends are supported:

- **Default (local)**: AES-256-GCM with a key derived from `JWT_SECRET` via HKDF-SHA256
- **AWS KMS**: When `MFA_KMS_KEY_ARN` is set, secrets are encrypted/decrypted via the KMS `Encrypt`/`Decrypt` API

The encryption interface is abstracted so additional backends (HashiCorp Vault, etc.) can be added.

### MFA Enforcement

When `server_settings.mfa_required` is `true`, the MFA enforcement middleware blocks all authenticated requests (except MFA setup endpoints) for users without `mfa_enabled`. This forces users to configure MFA before accessing any feature.

## Cursor on Target (CoT) Ingestion

SitAware can ingest Cursor on Target XML events, making it compatible with ATAK, iTAK, and other TAK ecosystem devices.

```
ATAK/iTAK Device
      │
      │  HTTP POST (CoT XML)
      ▼
POST /api/v1/cot/events
      │
      ├── Parse CoT XML (event UID, type, location, callsign, detail)
      ├── Store in cot_events table (spatial GEOMETRY column)
      └── Optionally link to user/device via event UID resolution
```

CoT events are stored with full XML detail for round-trip fidelity, along with parsed fields (location, callsign, event type, timestamps) for efficient querying. The `stale_time` field supports CoT's built-in event expiry semantics.

## Map Drawings

Users can create map annotations (lines, circles, rectangles) that are stored as GeoJSON and can be shared with group members.

```
Draw Panel (browser)
      │
      ├── Line: click to place vertices, double-click to finish
      ├── Circle: click center, drag radius
      └── Rectangle: click corner, drag to opposite corner
      │
      ▼
GeoJSON FeatureCollection
      │
      ├── Feature.properties: stroke, strokeWidth, fill, shape type
      └── Feature.geometry: LineString, Polygon (circle approximated as polygon)
      │
      ▼
POST /api/v1/drawings → stored in drawings table (JSONB)
      │
      ▼
Other clients receive via WebSocket → rendered as MapLibre layers
```

Each drawing is owned by a user and stored as a GeoJSON `FeatureCollection` in a JSONB column. Drawings can be shared to groups, and the drawing overlay component renders them as MapLibre sources/layers with per-feature styling.

## Terrain and Elevation

Terrain rendering is managed separately from map tiles via `terrain_configs`:

- Terrain sources provide elevation data (DEM tiles) in either `terrarium` or `mapbox` encoding
- The built-in default is AWS Terrarium tiles
- Admin can add custom terrain sources (e.g., self-hosted DEM tiles for air-gapped deployments)
- The map toggle enables/disables 3D terrain exaggeration on the MapLibre canvas
- Terrain configs support `remote` and `local` source types (local served from S3/Minio)

## Security Model

| Layer | Mechanism |
|---|---|
| Transport | TLS via reverse proxy (Caddy / ALB / Ingress) |
| Authentication | JWT access tokens (short-lived) + rotating refresh tokens |
| MFA | TOTP (authenticator apps), WebAuthn/FIDO2 (security keys, passkeys), recovery codes |
| Authorization | Role-based: admin, group_admin, member. Per-group permissions (can_read, can_write) |
| Password storage | bcrypt (cost 12) |
| Token storage | SHA-256 hashed refresh tokens in DB |
| TOTP secret storage | AES-256-GCM (HKDF-derived key) or AWS KMS envelope encryption |
| Rate limiting | Per-IP token bucket algorithm |
| Input validation | Request body size limits, handler-level validation |
| Audit | Every API action logged with user, resource, IP, timestamp |
| Container security | Distroless base image (API), non-root user (web), no shell in production containers |
