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

## API Layered Architecture

The API follows a strict layered design. Each layer only depends on the one below it.

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

**Middleware** (`internal/middleware/`) — Cross-cutting concerns applied to every request:

1. **CORS** — Validates `Origin` header against configured allowed origins
2. **MaxBodySize** — Wraps request body with `http.MaxBytesReader`
3. **RateLimit** — Per-IP token bucket rate limiting
4. **Logging** — Structured JSON request/response logging
5. **Audit** — Records API actions to the audit log
6. **Auth** — Per-route via `authMW.Authenticate()` or `authMW.RequireAdmin()`

**Handler** (`internal/handler/`) — One file per domain. Handlers parse HTTP input, delegate to a service, and write JSON responses. They never contain business logic or SQL.

**Service** (`internal/service/`) — Business rules, authorization checks, and cross-domain orchestration.

**Repository** (`internal/repository/`) — Pure data access using `pgx/v5` directly — no ORM.

**Model** (`internal/model/`) — Domain models, request/response DTOs, and a typed error system.

## Authentication

```
Login:
  Client → POST /api/v1/auth/login { username, password }
  Server → Verify password (bcrypt, cost 12)
         → Generate JWT access token (15min, HMAC-SHA256)
         → Generate opaque refresh token (crypto/rand UUID)
         → Store SHA-256(refresh_token) in DB with expiry
         → Return { access_token, refresh_token, user }

Token Refresh:
  Client → POST /api/v1/auth/refresh { refresh_token }
  Server → SHA-256 hash the token, look up in DB
         → Delete old token (rotation — each token is single-use)
         → Issue new access + refresh token pair
```

Refresh tokens are never stored in plaintext. Token rotation means a stolen refresh token can only be used once.

## Real-Time Architecture

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

Multiple API instances can run behind a load balancer. Redis pub/sub broadcasts messages across all nodes, enabling horizontal scaling without sticky sessions.

### Pub/Sub Channels

| Channel Pattern | Purpose |
|---|---|
| `group:{id}:location` | Real-time location updates for group members |
| `group:{id}:messages` | New messages in a group |
| `user:{id}:direct` | Direct messages to a specific user |
| `global:admin` | Admin-level notifications |

## Database Schema

PostgreSQL 16 with PostGIS for spatial operations. 17 tables across core, auth, MFA, messaging, spatial, and configuration domains.

```
users ──────< devices
  │
  ├────────< group_members >──────── groups
  │
  ├────────< messages ────────────── attachments
  │
  ├────────< location_history
  │
  ├────────< refresh_tokens
  │
  ├────────< audit_logs
  │
  ├────────< user_totp_methods
  │
  ├────────< webauthn_credentials
  │
  ├────────< recovery_codes
  │
  └────────< drawings

cot_events         (Cursor on Target — ATAK/iTAK ingestion)
map_configs        (tile source configurations)
terrain_configs    (terrain/elevation sources)
server_settings    (key-value server configuration)
```

Location fields use `GEOMETRY(POINT, 4326)` with GIST spatial indexes. All coordinates use WGS 84 (SRID 4326).

Migrations are embedded in the Go binary via `//go:embed` and run automatically on startup.

## Multi-Factor Authentication

Three MFA methods supported, all optional per-user unless the admin enables server-wide enforcement:

- **TOTP** — Authenticator apps (Google Authenticator, Authy, 1Password)
- **WebAuthn/FIDO2** — Security keys (YubiKey, Titan) and platform authenticators (Touch ID, Windows Hello)
- **Recovery Codes** — 8 one-time backup codes (bcrypt-hashed)

TOTP secrets are encrypted at rest using AES-256-GCM (HKDF-derived key from `JWT_SECRET`) or AWS KMS when `MFA_KMS_KEY_ARN` is set.

## Repository Structure

```
vincenty/
├── services/api/          # Go API service
│   ├── cmd/server/        # Entrypoint, DI, route registration
│   ├── internal/
│   │   ├── auth/          # JWT + password hashing
│   │   ├── config/        # Env-based configuration
│   │   ├── database/      # Migration runner + embedded SQL
│   │   ├── handler/       # HTTP handlers (one per domain)
│   │   ├── middleware/     # CORS, auth, logging, rate limit, audit
│   │   ├── model/         # Domain models + typed errors
│   │   ├── pubsub/        # Pub/sub interface + Redis impl
│   │   ├── repository/    # SQL queries (pgx)
│   │   ├── service/       # Business logic
│   │   ├── storage/       # Object storage interface + S3 impl
│   │   └── ws/            # WebSocket hub + client management
│   └── Dockerfile
├── clients/web/           # Next.js web client
├── clients/cli/           # CLI track streamer (Go)
├── clients/ios/           # iOS client (SwiftUI, XcodeGen)
├── deploy/
│   ├── caddy/             # Reverse proxy config (production)
│   ├── k8s/               # Kubernetes manifests
│   ├── helm/vincenty/     # Helm chart
│   └── ecs/               # AWS ECS Fargate task definitions
├── docker-compose.yml     # Development stack
├── docker-compose.prod.yml # Production stack (Caddy + TLS)
└── Makefile
```

## Security Model

| Layer | Mechanism |
|---|---|
| Transport | TLS via reverse proxy (Caddy / ALB / Ingress) |
| Authentication | JWT access tokens (short-lived) + rotating refresh tokens |
| MFA | TOTP, WebAuthn/FIDO2, recovery codes |
| Authorization | Role-based: admin, group_admin, member. Per-group permissions |
| Password storage | bcrypt (cost 12) |
| Token storage | SHA-256 hashed refresh tokens in DB |
| TOTP secret storage | AES-256-GCM or AWS KMS envelope encryption |
| Rate limiting | Per-IP token bucket algorithm |
| Audit | Every API action logged with user, resource, IP, timestamp |
| Container security | Distroless base image, non-root user, no shell |
