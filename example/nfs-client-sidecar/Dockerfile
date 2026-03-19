FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libnfs-utils \
    python3 \
    python3-libnfs \
    netcat-openbsd \
    bash \
    tini \
    && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /entrypoint.sh
COPY test_nfs.py /test_nfs.py
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/entrypoint.sh"]
