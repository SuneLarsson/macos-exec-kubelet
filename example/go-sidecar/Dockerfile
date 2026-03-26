## Build stage
FROM golang:1.21-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o /go-nfs-server .

## Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tini \
    && rm -rf /var/lib/apt/lists/*

# Create default export directory
RUN mkdir -p /nfsshare

COPY --from=builder /go-nfs-server /usr/local/bin/go-nfs-server

# NFS port (mountd + nfsd on same port)
EXPOSE 2049

USER 1000

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/usr/local/bin/go-nfs-server"]
