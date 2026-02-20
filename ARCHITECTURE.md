# Architecture

## System Overview

```
┌──────────────┐     ┌─────────────┐     ┌─────────────┐
│   Browser    │     │  ATAK/iTAK  │     │   Mobile    │
│  (Next.js)   │     │  (CoT XML)  │     │  (Future)   │
└──────┬───────┘     └──────┬──────┘     └──────┬──────┘
       │                    │                    │
       │  HTTP/WS           │  HTTP              │  HTTP/WS
       │                    │                    │
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

The system is composed of two application services (Go API, Next.js web client) backed by three infrastructure services (PostgreSQL with PostGIS, Redis, S3-compatible object storage). A reverse proxy (Caddy, ALB, or Ingress controller) sits in front for TLS termination and routing.

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

PostgreSQL 16 with PostGIS for spatial operations.

```
users ──────< devices
  │
  ├────────< group_members >──────── groups
  │
  ├────────< messages ────────────── attachments
  │              │
  │              └── (group_id OR recipient_id)
  │
  ├────────< location_history
  │
  ├────────< refresh_tokens
  │
  └────────< audit_logs
                 │
                 └── (optional group_id)

map_configs (standalone, admin-managed)
```

### Spatial Data

Location fields use `GEOMETRY(POINT, 4326)` with GIST spatial indexes. The `location_history` table stores every location update for replay. All coordinates use WGS 84 (SRID 4326).

### Migrations

Migrations are embedded in the Go binary via `//go:embed` and run automatically on startup using `golang-migrate`. Files are in `services/api/internal/database/migrations/` with the naming convention `{number}_{description}.{up|down}.sql`.

| Migration | Description |
|---|---|
| 000001 | Enable PostGIS and uuid-ossp extensions |
| 000002 | Create core tables (users, devices, groups, members, messages, attachments, locations, map_configs, audit_logs) |
| 000003 | Create indexes (spatial, temporal, foreign key) |
| 000004 | Create refresh_tokens table |
| 000005 | Create cot_events table (CoT support) |

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
| `/login` | Public | Login form |
| `/dashboard` | Authenticated | Overview, quick stats |
| `/map` | Authenticated | Real-time map with location markers, replay |
| `/messages` | Authenticated | Group and direct messaging |
| `/audit-logs` | Authenticated | User's own audit trail |
| `/admin/users` | Admin | User management |
| `/admin/groups` | Admin | Group and membership management |
| `/admin/map-configs` | Admin | Map tile source configuration |
| `/admin/audit-logs` | Admin | Full audit log viewer |

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

## Security Model

| Layer | Mechanism |
|---|---|
| Transport | TLS via reverse proxy (Caddy / ALB / Ingress) |
| Authentication | JWT access tokens (short-lived) + rotating refresh tokens |
| Authorization | Role-based: admin, group_admin, member. Per-group permissions (can_read, can_write) |
| Password storage | bcrypt (cost 12) |
| Token storage | SHA-256 hashed refresh tokens in DB |
| Rate limiting | Per-IP token bucket algorithm |
| Input validation | Request body size limits, handler-level validation |
| Audit | Every API action logged with user, resource, IP, timestamp |
| Container security | Distroless base image (API), non-root user (web), no shell in production containers |
