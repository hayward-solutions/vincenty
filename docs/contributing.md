# Contributing

## Development Setup

### Prerequisites

- [Go 1.23+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
- [Make](https://www.gnu.org/software/make/)

For iOS development (optional):

- macOS with [Xcode 16+](https://developer.apple.com/xcode/) (Swift 6.0, iOS 17 SDK)
- [XcodeGen](https://github.com/yonaskolb/XcodeGen) — `brew install xcodegen`

### Getting Started

```bash
git clone https://github.com/hayward-solutions/vincenty.git
cd vincenty
make dev
```

Web client at http://localhost:3000, API at http://localhost:8080. Default login: `admin` / `changeme`.

### Running Services Individually

```bash
# Start only infrastructure
make infra

# Run Go API locally
make api-dev

# Run Next.js dev server
make web-dev
```

### iOS Client Setup

```bash
cd clients/ios
xcodegen generate
open Vincenty.xcodeproj
```

Select the **Vincenty** scheme, choose a simulator, and run. The app needs the API running — enter `http://localhost:8080` when prompted.

## Code Conventions

### Go (API)

- Strict layered architecture: Handler → Service → Repository
- Go stdlib `net/http` with Go 1.22+ method-pattern routing
- Middleware pattern: `func(http.Handler) http.Handler`
- Typed errors in `model` package, translated to HTTP status codes in handlers
- `pgx/v5` for database access, no ORM
- All configuration via environment variables
- Run `go fmt` and `go vet` before committing

### TypeScript (Web)

- Next.js with App Router, server components by default
- UI components via [shadcn/ui](https://ui.shadcn.com/)
- All API calls through `src/lib/api.ts` (auto-refresh on 401)
- React Context for auth/WebSocket state, custom hooks for data fetching
- Tailwind CSS v4 utility classes

### Swift (iOS)

- MVVM with Observation framework (`@Observable`)
- All network calls through `APIClient.shared`
- Kebab-case file naming (e.g., `map-screen.swift`), PascalCase for models
- MapLibre Native SDK via UIViewRepresentable

## Adding a New Feature

### API — New Domain Resource

1. Add domain struct and DTOs to `internal/model/`
2. Add a migration in `internal/database/migrations/`
3. Create repository in `internal/repository/`
4. Create service in `internal/service/`
5. Create handler in `internal/handler/`
6. Register routes in `cmd/server/main.go`

### Web Client — New Page

1. Add TypeScript types to `src/types/api.ts`
2. Create data-fetching hook in `src/lib/hooks/`
3. Add page component under `src/app/(app)/`

### iOS Client — New Feature

1. Add Codable models to `Models/`
2. Create `@Observable` view model in `Features/{feature}/`
3. Create SwiftUI views, wire into navigation
4. Add `.accessibilityLabel` to icon-only buttons

## Pull Request Process

1. **Create an issue first** — discuss the approach before writing code
2. **Branch from `main`** — use descriptive names: `feature/...`, `fix/...`, `docs/...`
3. **Keep PRs focused** — one feature or fix per PR
4. **Test your changes** — `go build ./...`, `go vet ./...`, `make dev` must work
5. **Update documentation** — if you add config variables, endpoints, or change behavior
6. **Open the PR** — describe what and why, reference the issue, include screenshots for UI changes

## Code of Conduct

Be respectful and constructive. Focus on the work, not the person.
