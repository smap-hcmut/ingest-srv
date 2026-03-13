# syntax=docker/dockerfile:1

# =============================================================================
# STAGE 1: BUILDER
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -mod=vendor \
    -trimpath \
    -ldflags="-s -w" \
    -o consumer ./cmd/consumer

# =============================================================================
# STAGE 2: RUNTIME
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

LABEL org.opencontainers.image.source="https://github.com/smap-hcmut/dispatcher-srv" \
      org.opencontainers.image.description="SMAP Ingest Consumer" \
      org.opencontainers.image.licenses="MIT"

WORKDIR /app

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENV TZ=Asia/Ho_Chi_Minh

COPY --from=builder --chown=nonroot:nonroot /app/consumer .

USER nonroot:nonroot

ENTRYPOINT ["./consumer"]
