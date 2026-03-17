FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    nfs-ganesha \
    nfs-ganesha-vfs \
    dbus \
    bash \
    tini \
    libnss-wrapper \
    gettext-base \
    && rm -rf /var/lib/apt/lists/* \
    && rm -f /etc/mtab \
    && ln -s /proc/mounts /etc/mtab

# Create standard directories and set group permissions (GID 0) to allow rootless execution
RUN mkdir -p /nfsshare /var/run/dbus /var/lib/nfs/ganesha /etc/ganesha \
    && chgrp -R 0 /nfsshare /var/run/dbus /var/lib/nfs /etc/ganesha \
    && chmod -R g+rwX /nfsshare /var/run/dbus /var/lib/nfs /etc/ganesha

COPY entrypoint.sh /entrypoint.sh
COPY ganesha.conf.template /etc/ganesha/ganesha.conf.template

# Ensure the entrypoint script is executable and readable by the root group
RUN chgrp 0 /entrypoint.sh && chmod +x,g+rx /entrypoint.sh

# Expose only the unprivileged NFSv4 port
EXPOSE 2049

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/entrypoint.sh"]