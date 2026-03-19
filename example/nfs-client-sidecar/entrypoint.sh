#!/bin/bash
set -e

#todo

echo "--- Testing TCP connection to NFS Server ---"
echo "Target: ${NFS_SERVER}:${NFS_PORT}"
if nc -zv -w 5 "$NFS_SERVER" "$NFS_PORT"; then
    echo "SUCCESS: Network communication to the NFS pod is working!"
else
    echo "ERROR: Cannot reach NFS Server at ${NFS_SERVER}:${NFS_PORT}. This is a networking/routing issue!"
    sleep 3600
    exit 1
fi

echo ""
echo "--- Testing NFS protocol via userspace (without mount) ---"
echo "Attempting to list directory using nfs-ls..."
nfs-ls "nfs://${NFS_SERVER}:${NFS_PORT}${NFS_PATH}" || echo "Note: nfs-ls (libnfs) might fail if the server strictly enforces NFSv4 only, but the TCP connection test above confirms basic network reachability."

echo ""
echo "Client test finished. Keeping the pod alive for 1 hour."
sleep 3600
