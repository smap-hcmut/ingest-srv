# syntax=docker/dockerfile:1

# =============================================================================
# STAGE 1: BUILDER
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git tzdata

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -ldflags="-s -w" \
    -o api ./cmd/api

# =============================================================================
# STAGE 2: RUNTIME
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Ho_Chi_Minh

COPY --from=builder --chown=nonroot:nonroot /app/api .

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["./api"]
