FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY . .

RUN go mod tidy

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o cli-login ./cmd/main.go


FROM alpine:3.20

RUN apk add --no-cache libc6-compat ca-certificates

WORKDIR /app

COPY --from=builder /build/cli-login .
COPY migrations/ ./migrations/

RUN mkdir -p /app/data

ENV DB_PATH=/app/data/login.db

CMD ["./cli-login"]
