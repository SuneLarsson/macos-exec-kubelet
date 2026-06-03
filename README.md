# macos-exec-kubelet

A [Virtual Kubelet](https://github.com/virtual-kubelet/virtual-kubelet) provider that runs Kubernetes pods as native processes directly on a macOS Apple Silicon host — no VMs, no containers.

## What it does

When a Kubernetes Job or Pod is scheduled to this node, the kubelet:

1. Takes the `command` + `args` from the pod spec
2. Spawns them as a direct macOS process using `os/exec`
3. Reports process status (Running / Succeeded / Failed) back to Kubernetes
4. Streams stdout/stderr to log files accessible via `kubectl logs` (with automatic log rotation)
5. Automatically cleans up orphan processes from previous runs on startup

This gives you full access to host hardware — including the **Apple Neural Engine (ANE)**, Metal GPU, and any other Apple Silicon accelerators — from a standard Kubernetes workload, since the process runs natively on the host.

## When to use this

- You have a Mac Mini (or other Apple Silicon Mac) you want to schedule ML / inference workloads onto from a Kubernetes cluster
- You need direct hardware acceleration (CoreML, Metal, ANE) that is not available inside VMs
- You want simple process isolation without the overhead of virtualization

## How it differs from a normal kubelet

| Normal kubelet | macos-exec-kubelet |
|---|---|
| Runs pods in containers | Runs pods as host processes |
| Pulls container images | `image:` field is ignored — no image pull |
| Filesystem/network isolation | Shares host filesystem and network |
| Any OS | macOS / Apple Silicon only |

## Security: running as an unprivileged user

By default processes run as the same user as the kubelet. You can restrict jobs to a dedicated unprivileged user:

```bash
# One-time setup on the Mac (run as admin)
sudo bash scripts/create-runner-user.sh

# Then pass the username when starting the kubelet
./virtual-kubelet --runner-user vk-runner
```

The `vk-runner` account has no login shell and no home directory, but retains full access to system frameworks (CoreML, Metal, ANE).

## Prerequisites

- macOS on Apple Silicon (M1/M2/M3/M4)
- Go 1.22+ installed on the Mac
- An existing Kubernetes cluster with TLS certificates for a new node

## Building

Build directly on the Mac:

```bash
go build -o virtual-kubelet ./cmd/virtual-kubelet/
```

No codesigning is required for process execution.

## Running

```bash
export KUBECONFIG=/path/to/kubeconfig
export APISERVER_CERT_LOCATION=/path/to/kubelet.crt
export APISERVER_KEY_LOCATION=/path/to/kubelet.key
export APISERVER_CA_CERT_LOCATION=/path/to/ca.crt

nohup ./virtual-kubelet \
  --nodename mac-mini \
  --log-level info \
  --runner-user vk-runner \
  >> /var/log/virtual-kubelet.log 2>&1 &
```


## Writing a Job

Because there are no container images, your pod spec looks different from a typical Kubernetes Job. The `image` field is required by the API but is ignored at runtime. **You must always provide `command`.**

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: my-ml-job
  namespace: your-namespace
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: runner
        image: placeholder        # ignored — no image is pulled
        command: ["/bin/zsh", "-c", "python3 /Users/shared/train.py"]
        env:
        - name: MODEL_PATH
          value: /Users/shared/models/base
      tolerations:
      - key: "virtual-kubelet.io/provider"
        operator: "Exists"
        effect: "NoSchedule"
      nodeSelector:
        kubernetes.io/os: "darwin"
```

## Mounting Persistent Volumes (NFS)

macOS cannot natively mount Kubernetes PersistentVolumeClaims (PVCs) like CephFS or Longhorn. This provider works around that by mounting volumes over the network using NFS v3.

### How it works

An NFS server pod runs on a standard Linux worker node in your cluster, exports a directory (optionally backed by a PVC), and the kubelet mounts it on the Mac before the job starts.

1. **Deploy an NFS server pod** on a Linux node. It can export either:
   - A **PVC** (e.g. CephFS, Longhorn) mounted into the container — for persistent, shared storage.
   - An **emptyDir** or any local path — for ephemeral scratch space.
2. **Expose the pod with a NodePort Service** so the Mac can reach it.
3. **Annotate your macOS Job** to tell the kubelet where the NFS service is:

```yaml
  annotations:
    macos-exec-kubelet/nfs-service: "macos-nfs-service"
    macos-exec-kubelet/nfs-mount-path: "/proj/projectname"
```

Before the job process starts, the kubelet will:
- Wait for the NFS service endpoints to become ready.
- Look up the Node's InternalIP and the Service's NodePort.
- Execute the macOS native `mount -t nfs` command with NFS v3 options, mounting the share to a local path under `/private/tmp/<namespace>`.
- Automatically `umount` once the job completes or is deleted.

The mount options used by the kubelet are defined in [`pkg/nfsmount/nfsmount.go`](pkg/nfsmount/nfsmount.go). Currently they are:

```
vers=3,port=<nodePort>,mountport=<nodePort>,noresvport,noowners,rw,tcp
```

If you use a different NFS server that requires different options (e.g. NFS v4, separate mount port), edit `nfsmount.go` to match.

### Included NFS server

The repository includes a lightweight Go-based NFS server in [`example/go-sidecar/`](example/go-sidecar/) built on [github.com/willscott/go-nfs](https://github.com/willscott/go-nfs). It:

- Serves NFS v3 with both NFS and mount protocols on a single port (2049).
- Runs as non-root (UID 1000).
- Has been tested with CephFS PVCs for read and write operations.

This NFS server is **not mandatory** — you can swap it for any NFS server (Ganesha, the Linux kernel NFS server, etc.) as long as the mount options in `nfsmount.go` are adjusted to match.

### Example Workloads

The `example/` directory contains several deployable manifests demonstrating how to use the provider:

- **`exampleWithoutNFS.yaml`**: A simple "Hello world" job running directly on macOS without persistent storage.
- **`benchmarkExec.yaml`**: An AI benchmarking suite running on macOS, with data loaded over NFS.
- **`fioStorageTesting.yaml`**: A macOS storage benchmark running FIO over the NFS mount.
- **`fioLinuxTesting.yaml`**: A native Linux storage benchmark (for comparing native CephFS performance vs macOS NFS).

## Flags

| Flag | Default | Description |
|---|---|---|
| `--nodename` | hostname | Kubernetes node name |
| `--runner-user` | _(current user)_ | macOS user to run job processes as |
| `--log-level` | `info` | Log verbosity (`debug`, `info`, `warn`, `error`) |
| `--authentication-token-webhook` | `false` | Use Kubernetes TokenReview API to authenticate log/exec requests |
| `--authentication-token-webhook-cache-ttl` | | The duration to cache responses from the webhook token authenticator |
| `--authorization-webhook-cache-authorized-ttl` | | The duration to cache 'authorized' responses from the webhook authorizer |
| `--authorization-webhook-cache-unauthorized-ttl` | | The duration to cache 'unauthorized' responses from the webhook authorizer |
| `--client-verify-ca` | | CA cert to use to verify client requests |
| `--no-verify-clients` | `false` | Do not require client certificate validation |
| `--trace-sample-rate` | | Set probability of tracing samples |

## Project structure

```
cmd/virtual-kubelet/   # main entrypoint and CLI flags
pkg/
  client/              # ProcessClient — spawns and tracks processes
  provider/            # Virtual Kubelet provider interface implementation
  nfsmount/            # NFS mount/unmount logic (mount options configured here)
  event/               # Kubernetes event recording
  resource/            # Registry credential helpers (unused in exec mode)
internal/
  node/                # exec/attach I/O helpers
  netutil/             # network interface helpers for node IP reporting
  utils/               # Apple CPU model name sanitisation
scripts/
  create-runner-user.sh  # one-time macOS user setup
example/
  exampleWithoutNFS.yaml # Simple macOS job without NFS
  benchmarkExec.yaml     # macOS AI benchmark job with NFS server
  fioStorageTesting.yaml # macOS FIO benchmark job with NFS server
  fioLinuxTesting.yaml   # Linux native FIO benchmark job
  go-sidecar/            # Go-based NFS v3 server (container image source)
```

## Acknowledgements

This project is built on top of two open-source projects:

- **[virtual-kubelet](https://github.com/virtual-kubelet/virtual-kubelet)** — the CNCF framework that provides the Virtual Kubelet API and node lifecycle scaffolding that this provider plugs into.
- **[macOS-vz-kubelet](https://github.com/agoda-com/macOS-vz-kubelet)** by Agoda — a Virtual Kubelet provider that runs pods as macOS VMs using the Virtualization framework. This project started as a fork of that work; the VM execution layer was replaced with direct process execution (`os/exec`) to get native hardware access without virtualisation overhead.

## License

Apache 2.0
