# Features

## Authentication and User Management

### JWT Authentication
- Login with username and password
- Short-lived access tokens (15 minutes, HMAC-SHA256)
- Rotating opaque refresh tokens stored as SHA-256 hashes
- Single-use refresh tokens (rotation on every refresh)
- Automatic token refresh in the web client on 401 responses
- Logout with server-side token revocation
- Background cleanup of expired refresh tokens on a configurable interval

### User Management (Admin)
- Create, read, update, and delete user accounts
- Admin and regular user roles
- Last-admin protection (cannot delete or demote the only admin)
- Admin account bootstrapped from environment variables on first startup

### Device Management
- Users register devices (named endpoints — phones, tablets, radios)
- Devices inherit permissions from their owning user
- Update or remove devices at any time

## Groups and Permissions

### Groups
- Admin creates and manages groups
- Users are assigned to groups with granular permissions
- Each membership has `can_read`, `can_write`, and `is_group_admin` flags

### Permission Model
- **Admin** — full access to all resources, all groups, all users
- **Group Admin** — manage members within their groups, view group audit logs
- **Member (read)** — see group member locations and messages
- **Member (write)** — send messages and share location to the group
- Users only see data for groups they belong to (enforced at the service layer)

## Real-Time Location Tracking

### Location Sharing
- Browser Geolocation API captures the user's position
- Location updates sent via WebSocket to the API
- API writes to `location_history` table and publishes via Redis pub/sub
- All group members with read permission receive the update in real-time
- Configurable throttle interval (`WS_LOCATION_THROTTLE`) to control update frequency

### Location Display
- Real-time markers on the map for all visible group members
- Admin view: see all user locations across all groups
- User view: see locations of members in shared groups (respecting permissions)
- Marker labels show username and device name

### Location History and Replay
- Every location update is persisted with a timestamp
- Query location history for any user or group with time range filters
- Replay panel with time slider and playback controls
- Tracks rendered on the map during replay
- Export own location history as GPX files

## Mapping

### Map Display
- MapLibre GL JS with globe projection
- Configurable tile sources (admin-managed)
- Default map center and zoom level configurable via environment variables
- Layer controls for toggling overlays

### Map Configuration (Admin)
- Create and manage multiple tile source configurations
- Set the active tile source
- Support for local tile serving (tiles uploaded to S3/Minio)
- Custom tile URL templates with `{z}/{x}/{y}` placeholders

### GPX Support
- Upload GPX files as message attachments
- GPX tracks, routes, and waypoints rendered as overlays on the map
- Parse points, lines, and polygons from GPX XML

### Air-Gap Tile Serving
- Upload map tiles to the S3-compatible object store
- Configure `MAP_DEFAULT_TILE_URL` to point to the local tile endpoint
- Full map functionality with zero internet connectivity

## Messaging

### Group Messages
- Send text messages to any group you have write permission for
- Messages include sender's current location at time of send
- Real-time delivery to all online group members via WebSocket
- Persistent storage — scroll back through message history

### Direct Messages
- Send private messages to any other user
- Conversation list showing all active DM threads
- Real-time delivery via WebSocket

### File Attachments
- Attach files to any message (group or direct)
- Files stored in S3-compatible object storage
- Download attachments with token-authenticated URLs
- GPX files automatically parsed and renderable on the map

## Audit Logging

### Automatic Logging
- Every API action is recorded by the audit middleware
- Captured fields: timestamp, user, action, resource type, resource ID, IP address, user's location at time of action
- No manual instrumentation required — middleware captures everything

### Audit Log Access
- **Users** — view and export only their own audit logs
- **Group Admins** — view audit logs for members within their groups
- **Admins** — view and export complete audit logs for all users

### Export
- Export to CSV or JSON format
- Filterable by date range, user, action type

## WebSocket

### Connection Management
- Authenticated WebSocket connections (`GET /api/v1/ws?token=<jwt>`)
- Central Hub manages all active connections
- Per-user and per-group message routing
- Automatic subscription to relevant pub/sub channels based on group membership

### Message Types
- `location_update` — real-time position broadcasts
- `message` — new chat messages (group and direct)
- `ping/pong` — connection liveness checks

### Horizontal Scaling
- Redis pub/sub bridges WebSocket messages across multiple API instances
- No sticky sessions required — any client can connect to any API node

### Graceful Shutdown
- Hub sends WebSocket close frames to all connected clients on shutdown
- 2-second drain period ensures clients receive close notifications
- Write pumps exit cleanly via channel close (no context cancellation race)

## Security and Hardening

### Rate Limiting
- Per-IP token bucket algorithm using `golang.org/x/time/rate`
- Configurable requests-per-second and burst size
- Returns `429 Too Many Requests` with `Retry-After` header
- Stale rate limiter entries cleaned up periodically

### CORS
- Configurable allowed origins (comma-separated list)
- Validates `Origin` header and reflects the matched origin
- Sets `Vary: Origin` for proper caching behavior
- Supports `Access-Control-Allow-Credentials` for specific origins
- Wildcard `*` supported for development

### Request Size Limits
- Configurable maximum request body size (default 10MB)
- Applied globally via `http.MaxBytesReader` middleware
- Prevents memory exhaustion from oversized payloads

### Token Hygiene
- Expired refresh tokens purged on a configurable schedule
- Background goroutine runs cleanup without blocking request handling

### Container Security
- API runs in a distroless container — no shell, no package manager
- Web client runs as non-root `nextjs` user
- Multi-stage Docker builds minimize image size and attack surface

## Deployment

### Docker Compose (Development)
- Single command (`make dev`) starts the full stack
- PostgreSQL+PostGIS, Redis, Minio, API, Web
- Automatic Minio bucket creation via init container
- Hot reload: rebuild individual services with `make restart s=api`

### Docker Compose (Production)
- Caddy reverse proxy for TLS termination
- No exposed internal ports (only 80/443 via Caddy)
- Resource limits on all containers
- Redis password authentication enabled
- No Minio — uses external S3

### Kubernetes
- Full manifest set: namespace, configmap, secret, StatefulSet (PostgreSQL), deployments, services, Ingress
- API runs 2 replicas with readiness and liveness probes
- WebSocket-aware Ingress annotations
- TLS termination at the Ingress layer

### Helm Chart
- Fully parameterized via `values.yaml`
- Toggle between in-cluster and external PostgreSQL/Redis
- Configurable replica counts, resource limits, Ingress settings
- Template helpers for consistent labeling

### AWS ECS Fargate
- Fargate task definitions for API and web
- ALB for TLS and routing (including WebSocket support)
- Secrets from SSM Parameter Store (no plaintext credentials)
- IAM task roles for S3 access
- Service Connect for internal service discovery
- Deployment circuit breaker with automatic rollback
- CloudWatch log integration

### Air-Gapped Deployment
- Zero external dependencies at runtime
- No CDN calls, no external fonts, no analytics
- All UI assets bundled in the container image
- Map tiles served from local S3-compatible storage
- Works on a fully isolated network
