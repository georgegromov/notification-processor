FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.work go.work.sum ./
COPY shared ./shared
COPY notification-ingestor ./notification-ingestor
COPY notification-dispatcher ./notification-dispatcher
COPY producer ./producer

WORKDIR /src/notification-ingestor
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /bin/notification-ingestor ./cmd/notification-ingestor

FROM alpine:3.20

RUN apk add --no-cache ca-certificates
COPY --from=build /bin/notification-ingestor /usr/local/bin/notification-ingestor

ENTRYPOINT ["notification-ingestor"]
