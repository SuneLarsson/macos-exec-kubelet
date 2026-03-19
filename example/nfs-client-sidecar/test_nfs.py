import libnfs
import sys
import os

ip = sys.argv[1]
path = sys.argv[2]
# We connect to the root '/' pseudofs explicitly to bypass any CLI url-parsing bugs
url = f"nfs://{ip}/?version=4"

print(f"[Python NFS] Connecting to {url}")
try:
    nfs = libnfs.NFS(url)
except Exception as e:
    print(f"[Python NFS] Failed to connect: {e}")
    sys.exit(1)

print(f"[Python NFS] Connection successful! Listing root / :")
print(nfs.listdir("/"))

print(f"\n[Python NFS] Listing target directory {path} :")
try:
    print(nfs.listdir(path))
except Exception as e:
    print(f"[Python NFS] Failed to list {path}: {e}")
    sys.exit(1)

test_file = os.path.join(path, "test_from_python.txt")
print(f"\n[Python NFS] Attempting to write a file to {test_file} ...")
try:
    with nfs.open(test_file, mode='w') as f:
        f.write("Hello from userspace Python NFS client over NFSv4!\n")
    print(f"[Python NFS] Successfully wrote to {test_file}")
except Exception as e:
    print(f"[Python NFS] Failed to write file: {e}")
    sys.exit(1)

print(f"\n[Python NFS] Attempting to read the file back ...")
try:
    with nfs.open(test_file, mode='r') as f:
        content = f.read()
    print(f"[Python NFS] File content read successfully:")
    print("--------------------------------------------------")
    print(content)
    print("--------------------------------------------------")
except Exception as e:
    print(f"[Python NFS] Failed to read file: {e}")
    sys.exit(1)

print("[Python NFS] ALL TESTS PASSED!")
