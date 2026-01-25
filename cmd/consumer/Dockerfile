# syntax=docker/dockerfile:1

##
## STEP 1 - BUILD
##
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build deps if needed (gcc/musl-dev) — not required now, keep base lean

# Pre-cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build consumer binary
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -o consumer ./cmd/consumer

##
## STEP 2 - RUNTIME
##
FROM alpine:latest

ENV UID=2025 \
    USER=sefthost \
    GID=2025 \
    GROUP=sefthost

# Create non-root user
RUN addgroup -g "$GID" -S "$GROUP" \
    && adduser -D -S -h /app -s /bin/sh -G "$GROUP" -u "$UID" "$USER"

# Install certs + timezone data
RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates
ENV TZ=Asia/Ho_Chi_Minh
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /app
COPY --from=builder /app/consumer .

USER sefthost

ENTRYPOINT ["./consumer"]
