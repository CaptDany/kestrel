FROM golang:1.26-alpine AS builder
WORKDIR /build
RUN apk add --no-cache gcc musl-dev sqlite-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o kestrel -ldflags="-s -w" .
RUN CGO_ENABLED=0 go build -o scraper -ldflags="-s -w" ./cmd/scraper/

FROM alpine:3.21 AS kestrel
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 568 kestrel
USER kestrel
WORKDIR /home/kestrel
COPY --from=builder /build/kestrel .
EXPOSE 8000
VOLUME ["/home/kestrel/data"]
ENV KESTREL_PORT=8000 KESTREL_DB_PATH=/home/kestrel/data/kestrel.db
CMD ["./kestrel"]

FROM mcr.microsoft.com/playwright:v1.52.0-jammy AS scraper
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
RUN adduser --uid 568 --disabled-password --gecos "" kestrel
USER kestrel
WORKDIR /home/kestrel
COPY --from=builder /build/scraper .
EXPOSE 8001
ENV PORT=8001
CMD ["./scraper"]
