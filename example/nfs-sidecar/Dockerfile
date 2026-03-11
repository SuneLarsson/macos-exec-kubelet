FROM alpine:3.19

RUN apk add --no-cache \
    nfs-utils \
    bash \
    tini

# Create a default share directory
RUN mkdir -p /nfsshare

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Expose NFS port 2049, rpcbind 111, and mountd 20048
EXPOSE 2049 111 20048

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/entrypoint.sh"]
