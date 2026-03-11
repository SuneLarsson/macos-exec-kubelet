#!/bin/bash
set -eo pipefail

# The directory to export (can be overridden via ENV)
SHARED_DIRECTORY=${SHARED_DIRECTORY:-/nfsshare}

echo "Starting NFS server..."
echo "Exporting ${SHARED_DIRECTORY}"

# Ensure the shared directory exists
mkdir -p "${SHARED_DIRECTORY}"

# Write the exports file
# We export to '*' because access control is handled by the Kubernetes NetworkPolicy
echo "${SHARED_DIRECTORY} *(rw,sync,no_subtree_check,no_root_squash,insecure)" > /etc/exports

# Mount the rpc_pipefs if not already mounted
if ! mount | grep -q "rpc_pipefs"; then
  echo "Mounting rpc_pipefs..."
  mkdir -p /var/lib/nfs/rpc_pipefs
  mount -t rpc_pipefs rpc_pipefs /var/lib/nfs/rpc_pipefs
fi

# Start rpcbind (required for NFSv3/v4)
echo "Starting rpcbind..."
/sbin/rpcbind -w

# Start the NFS server natively
echo "Starting nfsd..."
/usr/sbin/exportfs -rv
/usr/sbin/rpc.nfsd
/usr/sbin/rpc.mountd
/usr/sbin/rpc.statd

echo "NFS server is running on port 2049."

# Keep the container alive by tailing the logs
exec tail -f /var/log/messages /var/log/syslog 2>/dev/null || while true; do sleep 3600; done
