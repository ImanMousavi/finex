FROM golang:1.16.4-alpine AS builder

WORKDIR /build

RUN set -ex &&\
    apk add --no-progress --no-cache \
      gcc \
      musl-dev

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOARCH="amd64" \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -tags musl -o finex-api ./cmd/finex-api/main.go
RUN go build -tags musl  -o finex-engine ./cmd/finex-engine/main.go
RUN go build -tags musl -o finex-daemon ./cmd/finex-daemon/main.go
RUN go build -tags musl -o finex-matching-engine ./cmd/finex-matching-engine/main.go


FROM alpine:3.13.6

RUN apk add ca-certificates
WORKDIR /app

COPY --from=builder /build/config/config.yaml ./config/config.yaml
COPY --from=builder /build/config/amqp.yml ./config/amqp.yml
COPY --from=builder /build/finex-api ./
COPY --from=builder /build/finex-engine ./
COPY --from=builder /build/finex-daemon ./
COPY --from=builder /build/finex-matching-engine ./
