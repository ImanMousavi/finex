FROM golang:1.18.1-alpine3.15 AS builder

WORKDIR /build

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOARCH="amd64" \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o finex-api ./cmd/finex-api/main.go
RUN go build -o finex-engine ./cmd/finex-engine/main.go
RUN go build -o finex-daemon ./cmd/finex-daemon/main.go
RUN go build -o finex-matching-engine ./cmd/finex-matching-engine/main.go


FROM alpine:3.13.6

RUN apk add ca-certificates
WORKDIR /app

COPY --from=builder /build/config/config.yaml ./config/config.yaml
COPY --from=builder /build/config/amqp.yml ./config/amqp.yml
COPY --from=builder /build/finex-api ./
COPY --from=builder /build/finex-engine ./
COPY --from=builder /build/finex-daemon ./
COPY --from=builder /build/finex-matching-engine ./
