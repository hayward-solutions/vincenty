# Contributing

## Development Setup

### Prerequisites

- [Go 1.23+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
- [Make](https://www.gnu.org/software/make/)

For iOS development (optional — only needed if working on `clients/ios/`):

- macOS (required for Xcode)
- [Xcode 16+](https://developer.apple.com/xcode/) (Swift 6.0, iOS 17 SDK)
- [XcodeGen](https://github.com/yonaskolb/XcodeGen) — `brew install xcodegen`
- Xcode Command Line Tools — `xcode-select --install`

### Getting Started

```bash
# Clone the repository
git clone https://github.com/vincenty/vincenty.git
cd vincenty

# Start the full stack
make dev
```

This builds and starts all services. The web client is at http://localhost:3000 and the API at http://localhost:8080. Default admin login: `admin` / `changeme`.

### Running Services Individually

If you prefer running the API or web client outside Docker (for faster iteration with your editor/debugger):

```bash
# Start only infrastructure (postgres, redis, minio)
make infra

# In one terminal: run the Go API locally
make api-dev

# In another terminal: run the Next.js dev server
make web-dev
```

When running the API locally, it connects to the Dockerized infrastructure using the defaults in `.env.example`. Copy `.env.example` to `.env` if you need to override anything.

### iOS Client Setup

The iOS client uses XcodeGen to generate the Xcode project from a `project.yml` spec. The only external dependency (MapLibre Native SDK) is fetched via Swift Package Manager automatically.

```bash
# Generate the Xcode project
cd clients/ios
xcodegen generate

# Open in Xcode
open Vincenty.xcodeproj
```

In Xcode, select the **Vincenty** scheme and an iOS Simulator (e.g., iPhone 16), then **Cmd+R** to build and run. The first build takes a few minutes while SPM fetches the MapLibre SDK.

The app needs the API running. On first launch, the server URL screen prompts for the API address:

- **Simulator**: `http://localhost:8080`
- **Physical device on same Wi-Fi**: `http://<your-mac-ip>:8080`

Log in with the default admin credentials (`admin` / `changeme`).

#### Testing on a physical device

1. You need an Apple Developer account (free tier works for personal testing)
2. In Xcode, go to **Signing & Capabilities** and select your team
3. If needed, change the bundle identifier to something unique (e.g., `com.yourname.vincenty`)
4. Connect your device via USB or Wi-Fi and run

Background location features require a real device — the simulator does not support CLLocationManager background modes.

#### Running iOS tests

```bash
cd clients/ios
xcodegen generate
xcodebuild test -scheme Vincenty -destination 'platform=iOS Simulator,name=iPhone 16'
```

### Useful Commands

```bash
make db-shell          # Open a psql shell to the database
make restart s=api     # Rebuild and restart just the API container
make clean             # Remove all containers and volumes (full reset)
make api-build         # Build the Go API binary locally
make cli-build         # Build the CLI binary locally
make cli-dev ARGS="--token=sat_... --file=track.gpx"  # Run CLI locally
```

## Project Structure

```
services/api/              # Go API service
  cmd/server/              # Entrypoint (main.go), connection helpers (db.go, redis.go, s3.go)
  internal/
    auth/                  # JWT generation/validation, password hashing
    config/                # Environment variable configuration
    database/              # Migration runner + embedded SQL files
    handler/               # HTTP handlers — one file per domain
    middleware/             # Request pipeline (CORS, auth, logging, rate limit, audit)
    model/                 # Domain models, DTOs, typed errors
    pubsub/                # Pub/sub interface + Redis implementation
    repository/            # Database queries (pgx)
    service/               # Business logic
    storage/               # Object storage interface + S3 implementation
    ws/                    # WebSocket hub, client, message types

clients/web/               # Next.js web client
  src/
    app/                   # App Router pages and layouts
    components/            # React components (ui/, map/, chat/, audit/)
    lib/                   # API client, auth context, custom hooks
    types/                 # TypeScript type definitions

clients/cli/               # CLI track streamer (Go)
  internal/
    client/                # REST + WebSocket client
    track/                 # GPX and GeoJSON parsers
  main.go                  # Entry point (flag + env var config)
  Dockerfile               # Distroless container image

clients/ios/               # iOS client (SwiftUI)
  Vincenty/
    App/                   # @main entry point, ContentView, MainTabView
    Models/                # Codable API models (User, Group, Message, etc.)
    Core/                  # Services (APIClient, AuthManager, WebSocket, SyncManager, etc.)
    Features/              # Feature modules (Auth, Map, Messages, Drawings, Settings, etc.)
    Components/            # Shared reusable UI components
    Extensions/            # Color+Hex, Date+Formatting
    Resources/             # Info.plist, entitlements, asset catalog
  project.yml              # XcodeGen project spec

deploy/                    # Deployment configurations
  caddy/                   # Caddyfile + TLS cert placeholder
  k8s/                     # Kubernetes manifests
  helm/vincenty/           # Helm chart
  ecs/                     # AWS ECS task definitions
```

## Code Conventions

### Go (API Service)

**Architecture** — The API uses a strict layered architecture: Handler -> Service -> Repository. Each layer only depends on the one below it. Never call a repository from a handler or a handler from a service.

**HTTP routing** — Uses Go stdlib `net/http` with Go 1.22+ method-pattern routing (`"GET /api/v1/users/{id}"`). No third-party router.

**Middleware** — Follows the stdlib pattern `func(http.Handler) http.Handler`. Global middleware is composed in `main.go`. Per-route auth middleware is applied at registration time.

**Error handling** — The `model` package defines typed errors (`ErrNotFound`, `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`, `ErrValidation`). Services return these errors; handlers translate them to HTTP status codes via `handler/response.go`.

**Database** — Uses `pgx/v5` directly. No ORM. Repositories accept `*pgxpool.Pool` and return domain models. SQL is written inline in repository methods.

**Configuration** — All via environment variables loaded in `internal/config/config.go`. Every variable has a default. Add new config fields to the struct and parse them in `Load()`.

**Dependencies** — Be conservative. The API has minimal dependencies by design. Discuss in the issue before adding new third-party packages.

**Formatting and linting** — Run `go fmt` and `go vet` before committing. The codebase follows standard Go conventions (effective Go, Go code review comments).

### TypeScript (Web Client)

**Framework** — Next.js with App Router. Pages use server components by default; add `"use client"` only when needed.

**UI components** — Built with [shadcn/ui](https://ui.shadcn.com/). Components live in `src/components/ui/`. Add new shadcn components with the CLI: `npx shadcn@latest add <component>`.

**API client** — All API calls go through `src/lib/api.ts`, which handles JWT attachment and auto-refresh. Do not use `fetch` directly in components.

**State management** — React Context for auth and WebSocket state. Custom hooks in `src/lib/hooks/` for data fetching. No external state library.

**Styling** — Tailwind CSS v4. Use utility classes directly. Avoid custom CSS files except for `globals.css`.

### Swift (iOS Client)

**Architecture** — MVVM with Swift Observation framework (`@Observable` macro). Views observe view models directly. Services (`APIClient`, `AuthManager`, `WebSocketService`, etc.) are injected via SwiftUI `.environment()`.

**Project generation** — The Xcode project is generated from `clients/ios/project.yml` using XcodeGen. Do not check in `Vincenty.xcodeproj` — regenerate it with `xcodegen generate`. The only external SPM dependency is MapLibre Native SDK.

**SwiftUI conventions** — iOS 17+ minimum. Files use **kebab-case** naming to match the web client convention (e.g., `map-screen.swift`). Exception: model files use **PascalCase** (e.g., `User.swift`). Components use **named exports** — `struct MapScreen: View`, no default exports. Add `"use client"` equivalent `@Observable` on view models and services.

**API models** — All API types live in `Models/`. Properties use `camelCase` in Swift; the shared `JSONDecoder`/`JSONEncoder` on `APIClient` converts to/from the API's `snake_case` via `.convertFromSnakeCase`/`.convertToSnakeCase` key strategies.

**API client** — All network calls go through the singleton `APIClient.shared`. Never use `URLSession` directly in views or view models. The client auto-refreshes JWT on 401 via `TokenManager` (actor-isolated, single-flight deduplication).

**Layered services** — `AuthManager` handles auth state, `WebSocketService` manages the real-time connection, `DeviceManager` handles device resolution, `LocationSharingManager` wraps `CLLocationManager`, `SyncManager` manages the offline queue. Views consume these via `@Environment`.

**Offline support** — SwiftData models in `Core/Storage/PersistentModels.swift` cache API data locally. `SyncManager` queues mutations when offline and drains them FIFO on reconnect with server-wins conflict resolution.

**Map** — MapLibre Native iOS SDK wrapped in `UIViewRepresentable`. Map controllers (`LocationMarkersController`, `DrawToolController`, `MeasureToolController`, etc.) are imperative classes that manage MapLibre sources and layers directly.

**Error handling** — `APIError` is the typed error from network calls. Views display errors in `ErrorBanner` or inline `Text` with `.foregroundStyle(.red)`. Never let raw errors propagate to the UI without a user-friendly message.

**Accessibility** — All icon-only buttons must have `.accessibilityLabel`. Interactive elements with visual state (selected, active) should use `.accessibilityValue`. Use `.accessibilityElement(children: .combine)` for composite elements. Test with VoiceOver on a real device.

**Testing** — Xcode test targets are defined in `project.yml`. Unit tests in `VincentyTests/`, UI tests in `VincentyUITests/`.

## Adding a New Feature

### API — New Domain Resource

1. **Model** — Add domain struct and DTOs to `internal/model/`
2. **Migration** — Add a new migration file in `internal/database/migrations/` (increment the number)
3. **Repository** — Create `internal/repository/{resource}_repo.go` with CRUD queries
4. **Service** — Create `internal/service/{resource}_service.go` with business logic and authorization
5. **Handler** — Create `internal/handler/{resource}_handler.go` with HTTP handlers
6. **Routes** — Register routes in `cmd/server/main.go` with appropriate auth middleware
7. **Wire** — Instantiate repo, service, and handler in `main.go` and pass dependencies

### Web Client — New Page

1. **Types** — Add TypeScript types to `src/types/api.ts`
2. **Hook** — Create a data-fetching hook in `src/lib/hooks/`
3. **Page** — Add the page component under `src/app/(app)/`
4. **Navigation** — Add link to the navigation layout if needed

### iOS Client — New Feature

1. **Types** — Add Codable model structs to `Models/` (mirror the API JSON shape, use `camelCase` properties)
2. **View Model** — Create an `@Observable` view model in `Features/{feature}/` with data fetching, mutation methods, and state
3. **Views** — Create SwiftUI views in `Features/{feature}/`. Use `@Environment` to access services
4. **Navigation** — Wire into `MainTabView` or the relevant `NavigationStack`
5. **Accessibility** — Add `.accessibilityLabel` to icon-only buttons, `.accessibilityValue` to stateful elements
6. **Offline** — If the feature supports offline, add a `CachedXxx` SwiftData model to `PersistentModels.swift` and queue mutations via `SyncManager.enqueue()`

### Database Migration

Migrations are numbered sequentially and embedded in the Go binary:

```
internal/database/migrations/
  000006_create_new_table.up.sql
  000006_create_new_table.down.sql
```

- `up.sql` — creates or alters tables
- `down.sql` — reverses the migration (DROP TABLE, etc.)
- Migrations run automatically on API startup
- Always test both up and down migrations locally

## Pull Request Process

1. **Create an issue first** — Describe the feature, bug, or improvement. Discuss the approach before writing code
2. **Branch from `main`** — Use a descriptive branch name: `feature/location-export`, `fix/ws-reconnect`, `docs/deployment-guide`
3. **Keep PRs focused** — One feature or fix per PR. If a change touches multiple domains, break it up
4. **Write clear commit messages** — Focus on *why*, not *what*. The diff shows what changed; the message explains the reasoning
5. **Test your changes**:
   - `go build ./...` and `go vet ./...` must pass
   - `make dev` must start the full stack without errors
   - Test the feature manually in the browser
6. **Update documentation** — If your change adds config variables, API endpoints, or changes behavior, update the relevant docs
7. **Open the PR** — Describe what changed and why. Reference the issue number. Include screenshots for UI changes

## Environment Variables

When adding a new environment variable:

1. Add it to `internal/config/config.go` (struct field + parsing in `Load()`)
2. Add it to `.env.example` with a sensible default and comment
3. Document it in `README.md` under the Configuration section
4. If it's a secret, add it to the Kubernetes secret template, Helm values, and ECS task definition secrets

## Database Schema Changes

- Never modify existing migration files that have been merged — always create a new migration
- Use `IF NOT EXISTS` / `IF EXISTS` guards where appropriate
- Add appropriate indexes for columns used in WHERE clauses or JOINs
- Use PostGIS `GEOMETRY(POINT, 4326)` for spatial columns with GIST indexes
- Test the down migration to ensure it cleanly reverses the up migration

## Deployment Changes

If your change affects deployment configurations, update all relevant targets:

| File | Purpose |
|---|---|
| `docker-compose.yml` | Development stack |
| `docker-compose.prod.yml` | Production Docker Compose |
| `deploy/k8s/*.yaml` | Kubernetes manifests |
| `deploy/helm/vincenty/values.yaml` | Helm chart values |
| `deploy/helm/vincenty/templates/*.yaml` | Helm templates |
| `deploy/ecs/*.json` | ECS task definitions |
| `deploy/ecs/README.md` | ECS deployment guide |

## Reporting Issues

When reporting a bug, include:

- Steps to reproduce
- Expected vs actual behavior
- Environment (OS, Docker version, browser)
- Relevant log output (`make logs` or browser console)

## Code of Conduct

Be respectful and constructive. Focus on the work, not the person. We're building something useful — keep discussions productive.
