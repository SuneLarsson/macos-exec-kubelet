#!/bin/bash

# The directory to export (can be overridden via ENV)
SHARED_DIRECTORY=${SHARED_DIRECTORY:-/nfsshare}

echo "Starting user-space NFS server (Ganesha) in containerization environment..."
echo "Exporting ${SHARED_DIRECTORY}"

# Ensure the shared directory exists
mkdir -p "${SHARED_DIRECTORY}" || true

export CURRENT_UID=$(id -u)
export CURRENT_GID=$(id -g)

# 1. Setup NSS Wrapper to map the random Kubernetes UID to a fake user named 'ganesha'
# This stops DBus from crashing with 'User unknown' errors
cp /etc/passwd /tmp/passwd
echo "ganesha:x:${CURRENT_UID}:${CURRENT_GID}:Ganesha User:/tmp:/bin/bash" >> /tmp/passwd
export NSS_WRAPPER_PASSWD=/tmp/passwd
export NSS_WRAPPER_GROUP=/etc/group
export LD_PRELOAD=libnss_wrapper.so

# Generate the config file from template using your sed logic
cat /etc/ganesha/ganesha.conf.template | sed \
  -e "s|\${SHARED_DIRECTORY}|${SHARED_DIRECTORY}|g" \
  -e "s|\${CURRENT_UID}|${CURRENT_UID}|g" \
  -e "s|\${CURRENT_GID}|${CURRENT_GID}|g" \
  > /tmp/ganesha.conf

# 2. Start an unprivileged DBus session to satisfy Ganesha's dependencies
echo "Starting unprivileged DBus session..."
export DBUS_SYSTEM_BUS_ADDRESS="unix:path=/tmp/dbus.sock"
dbus-daemon --session --address=$DBUS_SYSTEM_BUS_ADDRESS --fork || echo "dbus start failed"

# ganesha needs its run directory to write its pid file
mkdir -p /var/run/ganesha

# 3. Start Ganesha.nfsd
echo "Starting ganesha.nfsd..."
/usr/bin/ganesha.nfsd -F -L /tmp/ganesha.log -f /tmp/ganesha.conf
EXIT_CODE=$?

echo "Ganesha exited with code $EXIT_CODE"
echo "--- GANESHA LOGS ---"
cat /tmp/ganesha.log