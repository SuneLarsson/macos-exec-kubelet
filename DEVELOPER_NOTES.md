# Developer Notes: Execution Implementation

## Requirements
Ensure the following dependencies and tools are installed on your system:
* **Make** & **Xcode command-line tools**: Required for building and code signing.
* **Go 1.25.2**: Needs to be installed manually to ensure this specific version is used.


## Build Instructions
To build the execution implementation, run the following command from the project's root directory:

```bash
go build -o ../builds/macos-exec ./cmd/virtual-kubelet/
```

**Code Signing:**
Code signing shouldn't be strictly necessary if you are building and running the executable on the same machine, but it is good practice to apply it anyway:

```bash
codesign -s - --force --preserve-metadata=entitlements ../builds/macos-exec
```

## Connecting to Kubernetes
`admin.conf` was copied from one of the control-planes (with minor changes, such as the IP the server points to - either to a loadbalancer or directly to a control-plane). 
To make this work with the virtual-kubelet, the certificate data inside `admin.conf` was split out into separate files and mapped to environment variables:
```bash
export APISERVER_CA_CERT_LOCATION="$HOME/.kube/certs/ca.crt"
export APISERVER_CERT_LOCATION="$HOME/.kube/certs/tls.crt"
export APISERVER_KEY_LOCATION="$HOME/.kube/certs/tls.key"
export KUBECONFIG="$HOME/.kube/config"
```

*Security Note & Future Improvement:* Using `admin.conf` directly on the worker node is a significant security risk, as it grants full cluster admin privileges to the kubelet. Ideally, you should generate a dedicated kubelet client certificate, have it signed by the Kubernetes Certificate Authority (CA), and assign it narrowly scoped RBAC permissions instead.

## Running the Application
Start the service in the background using `nohup`. 

**Option 1: With a specific runner user (`vk-runner`)**
```bash
nohup sudo -E /path/to/builds/macos-exec --authentication-token-webhook true --nodename mac-mini-exec --log-level info --runner-user vk-runner > /path/to/logs/output.log 2>&1 &
```

**Option 2: Without a specific runner user**
```bash
nohup /path/to/builds/macos-exec --authentication-token-webhook true --nodename mac-mini-exec --log-level info > /path/to/logs/output.log 2>&1 &
```

## Namespace Configuration
When deploying, you need to ensure the appropriate tolerations are added to the `tolerationsWhitelist` (this might be handled automatically upon namespace creation, but it is worth verifying).

You need to apply the following toleration to allow pods to be scheduled:
```json
{
  "key": "virtual-kubelet.io/provider", 
  "operator": "Exists", 
  "effect": "NoSchedule"
}
```

**Targeting Specific Providers:**
If you have multiple Virtual Kubelet providers configured in your cluster (e.g., macOS and AWS) and want to guarantee that a workload is scheduled specifically on the Mac node, use the `Equal` operator instead:

```json
{
  "key": "virtual-kubelet.io/provider", 
  "operator": "Equal", 
  "value": "macos-exec", 
  "effect": "NoSchedule"
}
```
*(Note: adjust the `"value"` to match the exact provider name you are using).*
