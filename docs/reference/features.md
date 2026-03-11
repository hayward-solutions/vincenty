# Features

A complete reference of all Vincenty features.

## Authentication and User Management

- **JWT Authentication** — Short-lived access tokens (15min, HMAC-SHA256) with rotating opaque refresh tokens
- **API Tokens** — Long-lived `sat_` prefixed tokens for CLI and programmatic access
- **User Management** — Admin CRUD with last-admin protection
- **Device Management** — Register named devices, primary device flag, auto-detection via User-Agent
- **Multi-Factor Authentication** — TOTP (authenticator apps), WebAuthn/FIDO2 (security keys, passkeys), recovery codes
- **Passkey Login** — Passwordless authentication via WebAuthn discoverable credentials (web and iOS)
- **Server-wide MFA Enforcement** — Force all users to configure MFA
- **Avatar and Profile** — Upload profile pictures (JPEG, PNG, WebP up to 5 MB)
- **Marker Customization** — 10 shapes, 10 preset colors or custom hex for both users and groups

## Groups and Permissions

- Admin-managed groups with granular per-member permissions
- `can_read`, `can_write`, and `is_group_admin` flags per membership
- Role hierarchy: Admin > Group Admin > Member (read/write)
- Users only see data for groups they belong to

## Real-Time Location Tracking

- Browser Geolocation API captures position, sent via WebSocket
- Configurable throttle interval (`WS_LOCATION_THROTTLE`)
- Real-time markers on the map for all visible group members
- Location history with time range filters and replay controls
- Export own location history as GPX files

## Mapping

- **MapLibre GL JS** with globe projection and configurable tile sources
- **Admin-managed tile sources** — OpenStreetMap, Satellite (ESRI), custom, local
- **GPX support** — Upload GPX files, render tracks/routes/waypoints on the map
- **Drawing tools** — Lines, circles, rectangles with customizable stroke and fill colors
- **Measurement tools** — Distance and radius measurement with metric units
- **Terrain and 3D** — Toggle terrain rendering, admin-managed terrain sources
- **Filter panel** — Show/hide map layers and user markers
- **Air-gap tile serving** — Upload tiles to S3/Minio for offline operation
- **CoT/ATAK integration** — Ingest Cursor on Target XML events

## Messaging

- **Group messages** — Text messages with sender location, real-time delivery
- **Direct messages** — Private messaging between any two users
- **File attachments** — Attach files stored in S3, GPX files render on map

## Audit Logging

- Automatic logging of every API action (middleware-based, no instrumentation needed)
- Captured: timestamp, user, action, resource, IP, location
- Tiered access: users see own logs, group admins see group logs, admins see all
- Export to CSV or JSON with date/user/action filters

## WebSocket

- Authenticated connections (`GET /api/v1/ws?token=<jwt>`)
- Central Hub with per-user and per-group message routing
- Message types: `location_update`, `message`, `ping/pong`
- Horizontal scaling via Redis pub/sub (no sticky sessions)
- Redis Cluster mode supported for ElastiCache
- Graceful shutdown with close frames and drain period

## Security

- Per-IP token bucket rate limiting with `429 Retry-After`
- Configurable CORS with origin validation
- Request body size limits (default 10MB)
- Expired token cleanup on configurable schedule
- Distroless containers, non-root execution, multi-stage builds

## Deployment Options

- Docker Compose (development and production with Caddy TLS)
- Kubernetes (raw manifests)
- Helm chart (fully parameterized)
- AWS ECS Fargate (ALB, RDS, ElastiCache, SSM, IAM)
- Air-gapped deployment (zero internet dependency)
