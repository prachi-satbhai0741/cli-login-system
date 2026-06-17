# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

# gcc is required by go-sqlite3 (CGO)
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and compile
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o cli-login ./cmd/main.go


# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache libc6-compat ca-certificates

WORKDIR /app

# Copy the binary and migrations
COPY --from=builder /build/cli-login .
COPY migrations/ ./migrations/

# Data directory for the SQLite file (mounted as a volume)
RUN mkdir -p /app/data

ENV DB_PATH=/app/data/login.db

# Keep the container's stdin/tty open for interactive use
CMD ["./cli-login"]
