# macos-exec-kubelet

A [Virtual Kubelet](https://github.com/virtual-kubelet/virtual-kubelet) provider that runs Kubernetes pods as native processes directly on a macOS Apple Silicon host — no VMs, no containers.

## What it does

When a Kubernetes Job or Pod is scheduled to this node, the kubelet:

1. Takes the `command` + `args` from the pod spec
2. Spawns them as a direct macOS process using `os/exec`
3. Reports process status (Running / Succeeded / Failed) back to Kubernetes
4. Streams stdout/stderr to log files accessible via `kubectl logs`

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

### launchd (persistent daemon)

Drop a plist into `/Library/LaunchDaemons/com.thesis.virtual-kubelet.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>          <string>com.thesis.virtual-kubelet</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/virtual-kubelet</string>
    <string>--nodename</string>   <string>mac-mini</string>
    <string>--log-level</string>  <string>info</string>
    <string>--runner-user</string><string>vk-runner</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>KUBECONFIG</key>               <string>/var/root/.kube/config</string>
    <key>APISERVER_CERT_LOCATION</key>  <string>/etc/vk/kubelet.crt</string>
    <key>APISERVER_KEY_LOCATION</key>   <string>/etc/vk/kubelet.key</string>
    <key>APISERVER_CA_CERT_LOCATION</key><string>/etc/vk/ca.crt</string>
  </dict>
  <key>StandardOutPath</key>  <string>/var/log/virtual-kubelet.log</string>
  <key>StandardErrorPath</key><string>/var/log/virtual-kubelet.log</string>
  <key>RunAtLoad</key>  <true/>
  <key>KeepAlive</key>  <true/>
</dict>
</plist>
```

```bash
sudo launchctl load /Library/LaunchDaemons/com.thesis.virtual-kubelet.plist
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

## Flags

| Flag | Default | Description |
|---|---|---|
| `--nodename` | hostname | Kubernetes node name |
| `--runner-user` | _(current user)_ | macOS user to run job processes as |
| `--log-level` | `info` | Log verbosity (`debug`, `info`, `warn`, `error`) |
| `--authentication-token-webhook` | `false` | Use Kubernetes TokenReview API to authenticate log/exec requests |

## Project structure

```
cmd/virtual-kubelet/   # main entrypoint and CLI flags
pkg/
  client/              # ProcessClient — spawns and tracks processes
  provider/            # Virtual Kubelet provider interface implementation
  event/               # Kubernetes event recording
  resource/            # Registry credential helpers (unused in exec mode)
internal/
  node/                # exec/attach I/O helpers
  netutil/             # network interface helpers for node IP reporting
  utils/               # Apple CPU model name sanitisation
scripts/
  create-runner-user.sh  # one-time macOS user setup
example/
  job.yaml             # example Kubernetes Job
```

## Acknowledgements

This project is built on top of two open-source projects:

- **[virtual-kubelet](https://github.com/virtual-kubelet/virtual-kubelet)** — the CNCF framework that provides the Virtual Kubelet API and node lifecycle scaffolding that this provider plugs into.
- **[macOS-vz-kubelet](https://github.com/agoda-com/macOS-vz-kubelet)** by Agoda — a Virtual Kubelet provider that runs pods as macOS VMs using the Virtualization framework. This project started as a fork of that work; the VM execution layer was replaced with direct process execution (`os/exec`) to get native hardware access without virtualisation overhead.

## License

Apache 2.0
