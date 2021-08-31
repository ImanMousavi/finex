FROM golang:1.16.4-alpine AS builder

WORKDIR /build
ENV CGO_ENABLED=1 \
  GOOS=linux \
  GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o finex-api ./cmd/finex-api/main.go
RUN go build -o finex-engine ./cmd/finex-engine/main.go
RUN go build -o finex-daemon ./cmd/finex-daemon/main.go


FROM alpine:3.12.7

RUN apk add ca-certificates
WORKDIR /app

COPY --from=builder /build/config/config.yaml ./config/config.yaml
COPY --from=builder /build/config/amqp.yml ./config/amqp.yml
COPY --from=builder /build/finex-api ./
COPY --from=builder /build/finex-engine ./
COPY --from=builder /build/finex-daemon ./
