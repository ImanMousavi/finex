FROM golang:1.16.4-alpine AS builder

RUN apk add --no-cache curl

ARG KAIGARA_VERSION=0.1.24
# Install Kaigara
RUN curl -Lo /usr/bin/kaigara https://github.com/openware/kaigara/releases/download/${KAIGARA_VERSION}/kaigara \
  && chmod +x /usr/bin/kaigara

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


FROM alpine:3.9

RUN apk add ca-certificates
WORKDIR app
COPY --from=builder /build/finex-api ./
COPY --from=builder /usr/bin/kaigara /usr/bin/kaigara
