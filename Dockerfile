# =============================================================================
# Stage 1 — Builder
# =============================================================================
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 — статическая сборка, финальный образ минимален
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/processor ./cmd/notification-processor

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/producer ./cmd/event-producer

# =============================================================================
# Stage 2 — Final image
# =============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/processor .
COPY --from=builder /bin/producer .
COPY migrations/schema.sql migrations/schema.sql

CMD ["./processor"]