# Stage 1: Builder
FROM golang:1.27-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-orchestrator ./cmd/raggo-orchestrator && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-rss-podcast ./cmd/raggo-rss-podcast && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-download-audio ./cmd/raggo-download-audio && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-stt-audio ./cmd/raggo-stt-audio && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-normalize-podcast ./cmd/raggo-normalize-podcast && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-chunk-podcast ./cmd/raggo-chunk-podcast && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-contextualize-podcast ./cmd/raggo-contextualize-podcast && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-embed-podcast ./cmd/raggo-embed-podcast && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-index-podcast ./cmd/raggo-index-podcast && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-scan-documents ./cmd/raggo-scan-documents && \
    CGO_ENABLED=1 GOOS=linux go build -o /bin/raggo-extract-text ./cmd/raggo-extract-text && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-chunk-text ./cmd/raggo-chunk-text && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-contextualize-text ./cmd/raggo-contextualize-text && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-embed-text ./cmd/raggo-embed-text && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/raggo-index-documents ./cmd/raggo-index-documents

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/raggo-* /usr/local/bin/

RUN mkdir -p /var/lib/raggo /var/log/raggo /etc/raggo

WORKDIR /app

ENV RAGGO_CONFIG=/etc/raggo/raggo.yml

ENTRYPOINT ["raggo-orchestrator"]
CMD ["-config", "/etc/raggo/raggo.yml"]
