FROM golang:1.27-trixie AS builder       
# already includes gcc and git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ./scripts/build.sh /out

FROM debian:trixie-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /out/ /usr/local/bin/
RUN mkdir -p /var/lib/raggo /var/log/raggo /etc/raggo
WORKDIR /app
ENV RAGGO_CONFIG=/etc/raggo/raggo.yml
ENTRYPOINT ["raggo-orchestrator"]
CMD ["-config", "/etc/raggo/raggo.yml"]