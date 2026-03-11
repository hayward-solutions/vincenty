# =============================================================================
# Build stage
# =============================================================================
FROM golang:alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
ARG GIT_COMMIT=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${GIT_COMMIT}" -o /build/server ./cmd/server

# =============================================================================
# Runtime stage
# =============================================================================
FROM gcr.io/distroless/static-debian12

COPY --from=builder /build/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
