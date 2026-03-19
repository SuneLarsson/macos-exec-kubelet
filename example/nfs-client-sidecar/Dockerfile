FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libnfs-utils \
    netcat-openbsd \
    bash \
    tini \
    && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/entrypoint.sh"]
