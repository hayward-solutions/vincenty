# AGENTS.md

## Project Overview

Monorepo with a Go API (`services/api/`) and Next.js web client (`clients/web/`).
Go 1.25, Node 20, PostgreSQL+PostGIS, Redis, S3/Minio. Strict layered architecture:
Handler → Service → Repository. See `ARCHITECTURE.md` for full system design.

## Build / Lint / Test Commands

### Go API (`services/api/`)

```bash
# Build
cd services/api && CGO_ENABLED=0 go build -o bin/server ./cmd/server

# Format and vet (run before every commit)
cd services/api && go fmt ./... && go vet ./...

# Run all unit tests (no Docker needed)
cd services/api && go test ./internal/...

# Run a single test by name
cd services/api && go test ./internal/handler/ -run TestHandleError

# Run a single test file's package
cd services/api && go test ./internal/model/

# Run integration tests (requires Docker daemon running)
cd services/api && go test ./internal/integration/...

# Verbose integration test output
cd services/api && TEST_VERBOSE=1 go test -v ./internal/integration/...

# Run all tests
cd services/api && go test ./...
```

### Next.js Web Client (`clients/web/`)

```bash
# Install dependencies
cd clients/web && npm ci

# Dev server
cd clients/web && npm run dev

# Production build (must pass in CI)
cd clients/web && npm run build

# Lint (ESLint 9 flat config, next core-web-vitals + typescript)
cd clients/web && npm run lint

# Run all tests (Vitest)
cd clients/web && npm test

# Run a single test file
cd clients/web && npx vitest run src/lib/hooks/__tests__/use-users.test.ts

# Run tests matching a name pattern
cd clients/web && npx vitest run -t "should fetch users"

# Watch mode
cd clients/web && npm run test:watch

# Coverage
cd clients/web && npm run test:coverage
```

### Docker / Full Stack

```bash
make dev           # Start all services (docker compose up --build)
make infra         # Start only postgres, redis, minio
make api-dev       # Run Go API locally (needs infra)
make web-dev       # Run Next.js dev server locally
make down          # Stop all services
make clean         # Remove all containers and volumes
```

## Code Style — Go API

### Architecture Rules
- **Handler → Service → Repository** — never skip layers. Handlers must not import repositories. Services must not import `net/http`.
- Constructors: `NewFooHandler(svc *service.FooService) *FooHandler`.
- Receivers: single-letter (`h` handler, `s` service, `r` repository).

### Imports
Two groups separated by a blank line (standard `goimports` format):
```go
import (
    "context"
    "net/http"

    "github.com/google/uuid"
    "github.com/sitaware/api/internal/model"
)
```
Stdlib first, then third-party + internal combined. No third blank-line group.

### Error Handling
- Six typed errors in `internal/model/errors.go`: `ErrValidation`, `ErrNotFound`, `ErrConflict`, `ErrForbidden`, `ErrMFARequired`, `ErrMFASetupRequired`.
- Services return these typed errors; handlers dispatch via `HandleError(w, err)` in `handler/response.go`.
- Repositories translate `pgx.ErrNoRows` → `model.ErrNotFound("resource")` immediately — never leak pgx errors.
- Use `errors.As(err, &target)` for type checking, not direct type assertions.
- Intentionally ignored errors must be explicit: `_ = s.storageSvc.Delete(...)`.

### Naming
- Files: `{noun}_{layer}.go` — `user_handler.go`, `user_service.go`, `user_repo.go`.
- Structs: `PascalCase`. Request DTOs: `CreateUserRequest`. Response DTOs: `UserResponse`.
- JSON tags: `snake_case` — `json:"access_token"`. Internal fields: `json:"-"`.
- Never serialize `User` directly; convert via `.ToResponse()` to `UserResponse`.

### Database
- `pgx/v5` with `*pgxpool.Pool`. Raw SQL only — no ORM, no query builder.
- Parameterized queries with `$1, $2` placeholders. Multi-line SQL in backtick strings.
- UUID generation happens in the repository if `uuid.Nil`.

### Logging
- `log/slog` (stdlib) only. JSON handler in production, discard in tests.
- Pattern: `slog.Error("message", "error", err, "key", value)`.

### Testing
- Stdlib `testing` package only — no testify, no gomock.
- Unit tests: same package, `*_test.go` alongside source files.
- Integration tests: `internal/integration/` using `testcontainers-go` (real Postgres/Redis/Minio).
- Test helpers in `internal/testutil/testutil.go`: `TestEnv`, `DoJSON`, `RequireStatus`, `LoginAdmin`.

## Code Style — Next.js Web Client

### Component Conventions
- Files: **kebab-case** — `message-bubble.tsx`, `map-view.tsx`.
- Components: **named exports only** — `export function MapView(...)`. No default exports except App Router pages/layouts.
- Props: inline `interface FooProps {}` above the component, not exported unless shared.
- `"use client"` directive on all interactive components and hooks. Server components omit it.
- shadcn/ui components in `src/components/ui/`. Add new ones via `npx shadcn@latest add <name>`.

### Imports
Order (no automated sorter — follow manually):
1. `"use client"` directive (first line when needed)
2. React / Next.js imports
3. Third-party libraries
4. `@/lib/api`, `@/lib/*-context`
5. `@/lib/hooks/*`
6. `@/components/*`
7. `@/types/api` — always use `import type` for type-only imports

### Types
- All API types in a single file: `src/types/api.ts`.
- Properties use `snake_case` (mirrors Go JSON output).
- Generic list wrapper: `ListResponse<T>` with `data`, `total`, `page`, `page_size`.
- WebSocket types prefixed `WS*`. Request types suffixed `Request`. Response types suffixed `Response`.

### API Client
- Singleton `api` instance in `src/lib/api.ts`. All API calls go through it.
- Never use `fetch()` directly in components or hooks.
- Throws `ApiError(status, message)` on non-OK responses. Check with `err instanceof ApiError`.
- Auto-refreshes JWT on 401 and retries once.

### Hooks
- Data hooks return `{ data, isLoading, error, refetch }`.
- Mutation hooks return `{ actionFn, isLoading }` — errors propagate to caller.
- WebSocket hooks use `subscribe()` from `WebSocketContext`.

### Styling
- Tailwind CSS v4. Utility classes only — no custom CSS files besides `globals.css`.
- `cn()` helper from `src/lib/utils.ts` (`twMerge(clsx(...))`).
- Variants via `cva()` from `class-variance-authority`.
- Dark mode via `.dark` class. Colors in oklch.

### Testing
- Vitest + `@testing-library/react` + MSW for API mocking.
- Test files in `__tests__/` subdirectories: `src/lib/hooks/__tests__/use-users.test.ts`.
- Shared fixtures in `src/test/fixtures.ts`. MSW handlers in `src/test/msw-handlers.ts`.
- Test utilities in `src/test/test-utils.tsx` (mocked auth/websocket contexts, custom `render`).
- Hook tests use `renderHook` + `waitFor`. Override API responses with `server.use(http.get(...))`.
- **Important**: tests of `AuthProvider` itself must NOT import `test-utils.tsx` (it mocks auth-context).

## Adding a New Feature

### API side
1. Model + DTOs in `internal/model/`
2. Migration in `internal/database/migrations/` (next sequential number)
3. Repository in `internal/repository/{resource}_repo.go`
4. Service in `internal/service/{resource}_service.go`
5. Handler in `internal/handler/{resource}_handler.go`
6. Routes in `cmd/server/main.go` with auth middleware
7. Wire repo → service → handler in `main.go`

### Web client side
1. Types in `src/types/api.ts`
2. Hook in `src/lib/hooks/use-{resource}.ts`
3. Page in `src/app/(app)/`
4. Tests for new hooks in `src/lib/hooks/__tests__/`

### Feature Parity
When adding features, ensure both the API and web client are updated together. If
a new endpoint is added, add the corresponding types, hooks, and UI. Update tests
on both sides. Keep `ARCHITECTURE.md` and `CONTRIBUTING.md` current if the change
affects project structure or conventions.

## CI

PR checks (`.github/workflows/pull_request.yaml`): Go build + Next.js build.
Only runs for changed paths (`services/api/**` or `clients/web/**`).
Merge to main (`.github/workflows/merge.yaml`): builds + publishes Docker images to GHCR.
