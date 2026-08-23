# syntax=docker/dockerfile:1.7

# To pin base images for reproducible builds, replace tags with digests:
#   docker pull golang:1.26-alpine
#   docker image inspect --format '{{index .RepoDigests 0}}' golang:1.26-alpine
# Then use: FROM golang:1.26-alpine@sha256:<digest> AS builder

FROM golang:1.27-alpine AS builder
WORKDIR /build
RUN apk add --no-cache gcc musl-dev sqlite-dev
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -o kestrel -ldflags="-s -w" .

FROM golang:1.27-alpine AS scraper-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o scraper -ldflags="-s -w" ./cmd/scraper/

FROM alpine:3.24 AS kestrel
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 568 kestrel
USER kestrel
WORKDIR /home/kestrel
COPY --from=builder /build/kestrel .
EXPOSE 8000
VOLUME ["/home/kestrel/data"]
ENV KESTREL_PORT=8000 KESTREL_DB_PATH=/home/kestrel/data/kestrel.db
CMD ["./kestrel"]

FROM mcr.microsoft.com/playwright:v1.62.1-jammy AS scraper
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
RUN adduser --uid 568 --disabled-password --gecos "" kestrel
USER kestrel
WORKDIR /home/kestrel
COPY --from=scraper-builder /build/scraper .
EXPOSE 8001
ENV PORT=8001
CMD ["./scraper"]
