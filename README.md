# macOS Virtualization Kubelet - Run native macOS workloads on Kubernetes

`macOS-vz-kubelet` bridges the worlds of Kubernetes and native macOS workloads. It enables macOS hosts to act as Kubernetes nodes, allowing you to deploy and manage macOS Virtual Machines at scale. The project also supports running Docker containers alongside macOS VMs within the same Pod, providing flexibility for hybrid workloads.

See [examples](example) directory for pod manifests, such as a [macOS VM pod](example/pod.yml) and a [hybrid pod with macOS VM and Docker side-car container](example/pod-gitlab-sidecar.yml).

## Introduction

`macOS-vz-kubelet` integrates native macOS workloads into Kubernetes by leveraging Apple's [Virtualization framework](https://developer.apple.com/documentation/virtualization) and the [Virtual Kubelet](https://github.com/virtual-kubelet/virtual-kubelet) project. Unlike traditional solutions that rely on QEMU/KVM under Linux, which struggle to fully utilize Apple Silicon's performance, `macOS-vz-kubelet` enables macOS hosts to operate as Kubernetes nodes with near-native performance.

This project is designed to:

- Run macOS Virtual Machines as first-class citizens in a Kubernetes cluster.

- Support hybrid Pods that combine macOS VMs with side-car containers.

- Streamline VM image management with a custom OCI-compliant format.

- Provide Kubernetes integration for scheduling and resource management.

`macOS-vz-kubelet` is an ideal solution for teams looking to scale macOS workloads, such as CI/CD pipelines or testing environments, while maintaining compatibility with Kubernetes' extensive feature set. For a detailed list of supported and unsupported features, see the [Feature Overview](#feature-overview).

## How It Works

`macOS-vz-kubelet` transforms macOS hosts into Kubernetes nodes by utilizing Apple’s Virtualization framework and acting as a Virtual Kubelet provider. This allows Kubernetes to orchestrate native macOS workloads with near-native performance, alongside optional Docker-based containers for hybrid workloads.

### Key Components

1. **Virtualization Framework**

   Apple’s Virtualization framework provisions and manages macOS Virtual Machines (VMs) natively. This ensures optimal performance by directly leveraging Apple Silicon hardware.

1. **Custom OCI Format for VM Images**

   The project introduces a custom OCI-compliant image format to manage VM images efficiently. Images are created with Virtualization.framework compliant tools and distributed using our forked ORAS CLI ([oras-macos-vz](https://github.com/agoda-com/oras-macos-vz)) tailored for this project.

1. **Hybrid Runtime Pods**

   Each Pod’s first container is always a macOS VM.
Side-car containers, managed by the Docker runtime, can complement the VM for tasks like logging, monitoring, or artifact management.

1. **Networking**

   Networking is managed automatically by macOS-vz-kubelet, with support for both local NAT mode and bridged networking for more complex setups. See [Networking](#networking) section for more detailed explanation.

1. **Resource and Lifecycle Management**

   Resource requests (CPU, memory) are supported for macOS VMs.

   Pod lifecycle events like creation and deletion are fully supported, while updates require pod recreation.

### Workflow

1. A Kubernetes Pod manifest specifies the VM image in the custom OCI format.

1. The Virtual Kubelet retrieves the image from the OCI registry.

1. The Virtualization framework provisions the macOS VM. Optional side-car Docker containers are started alongside.

1. The Virtual Kubelet updates the Kubernetes control plane with the Pod’s status, IPs, and lifecycle events.

For detailed steps on setting up and running workloads, see the [Usage Guide](#usage-guide).

## Networking

Networking in `macOS-vz-kubelet` can operate in two modes, depending on workload requirements:

1. NAT Mode (Default)

   - VMs are assigned local IP addresses via NAT.

   - Suitable for most use cases where external IP access is unnecessary.

   - Interaction with the VM is managed through kubectl exec, which uses SSH under the hood to connect locally.

1. Bridged Networking

   - Enables remote IP access by connecting VMs to tagged VLANs via DHCP.

   - Custom MAC addresses are assigned to each VM interface to ensure IP tracking.

   - IPs are dynamically retrieved by the `macOS-vz-kubelet` using tools like tcpdump and reported back to Kubernetes.

For Bridged Networking VMNet and VM Networking capabilities are required. These 2 capabilities require Apple's approval. Follow [this Apple Forum thread](https://developer.apple.com/forums/thread/656411) on how to request it. After that just pass desired network interface name to `VZ_BRIDGE_INTERFACE` environment variable.

Afterwards, simply generate yourself Mac Development certificate, App ID with those capabilities, and provision profile. Input those in Makefile and enjoy.

## Imaging

As mentioned before, the project introduces a custom OCI-compliant image format to manage VM images efficiently. See [Setup Workflow](#setup-workflow) for detailed steps on creating and pushing VM images to the registry. You can also check [OCI manifest example](example/oci_manifest.json) of our format.

Below are some key points about the imaging process.

### Compression

We we are running compression during the image packaging into OCI. The reason for that is quite simple. On average, our current macOS images are way above ~55 Gigabytes with tools like Xcode and simulators pre-installed. While we don't have to update them often, we still prefer to downsize them as much as possible before being able to distribute them. Using our own OCI content store implementation with custom compression, we can maintain our images on average at the ~35-gigabyte mark in our company's registry.

### Copy-on-Write (COW)

Each host can run two Virtual Machines (Pods) simultaneously (this number is a limitation of Virtualization framework). To avoid conflicts and ensure scalability we use a [copy-on-write (COW)](https://github.com/apple/darwin-xnu/blob/main/bsd/sys/clonefile.h) mechanism to create overlays of the disk image for each Pod. VMs on the same host can share the base image while maintaining their independent state through overlay files.

### Digest validation

We maintain a calculated digest for local image files to guarantee the correctness and integrity of VM images. The process is as follows:

- When a VM image is first downloaded from the remote OCI registry, a digest is generated for the .img file.

- Each time a Pod starts, the calculated digest is compared to the recorded digest from the remote store.

- If the digest is missing or if the .img file is newer than the digest file, it indicates that the local cache is invalid, and the image is re-downloaded from the remote OCI registry.

## Feature Overview

`macOS-vz-kubelet` supports the following Kubernetes features. Features not listed below are currently unsupported.

### Node

| Feature                                  | Supported | Comments           |
|------------------------------------------|:---------:|--------------------|
| **Node addresses**                       | ✅        |                    |
| **Node capacity**                        | ✅        |                    |
| **Node daemon endpoints**                | ✅        |                    |
| **Operating system**                     | ✅        | Darwin macOS only. |

### Pod

| Feature                                  | Supported | Comments                                                                                                                                           |
|------------------------------------------|:---------:|----------------------------------------------------------------------------------------------------------------------------------------------------|
| **Create and delete pods**               | ✅        |                                                                                                                                                    |
| **Update pods**                          | ❌        | Recreate the pod instead.                                                                                                                          |
| **Get pod, pods and pod status**         | ✅        |                                                                                                                                                    |
| **Security policies**                    | ❌        |                                                                                                                                                    |
| **Init containers**                      | ❌        | On the short list.                                                                                                                                 |
| **Regular containers**                   | ✅        | Supported using docker client. First container on the pod must always be macOS VM, every next one is supported as a regular (docker) container.    |

### Containers

| Feature                                  | Supported | Comments                                                                                                                                                                                                          |
|------------------------------------------|:---------:|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Container logs**                       | ⚠️         | Only for docker containers.                                                                                                                                                                                       |
| **Container exec**                       | ✅        | `VZ_SSH_USER` and `VZ_SSH_PASSWORD` env variables must be set and correspond to macOS VM ssh user and password in order for exec into macOS containers to work. Exec into the regular container works by default. |
| **Container attach**                     | ⚠️         | Supported, but not tested.                                                                                                                                                                                        |
| **Container metrics**                    | ❌        |                                                                                                                                                                                                                   |
| **Resource requests**                    | ⚠️         | MacOS VMs are created with these resource definitions. Docker containers do not support this feature.                                                                                                             |
| **Resource limits**                      | ❌        | Generally ignored due to VM nature.                                                                                                                                                                               |
| **Health checks (liveness, readiness)**  | ❌        |                                                                                                                                                                                                                   |

### Storage

| Feature                                  | Supported | Comments                                                                                                                                                                                                          |
|------------------------------------------|:---------:|--------------------------------------------|
| **Host volumes**                         | ✅        |                                            |
| **Empty dir volumes**                    | ✅        |                                            |
| **Persistent volumes**                   | ❌        | Unsupported by virtual kubelet in general. |
| **Config maps volumes**                  | ❌        | On the short list.                         |
| **Secrets volumes**                      | ❌        | On the short list.                         |
| **Projected volumes**                    | ⚠️         | See the table below.                       |

A [projected volumes](https://kubernetes.io/docs/concepts/storage/projected-volumes) map several existing volume sources into the same directory.

By default, Kubernetes adds a projected volume mount with a service account token, api server key and namespace name that can be used to call k8s API server from the containers in the pod.

| Feature                   | Supported | Comments                                                         |
|---------------------------|:---------:|------------------------------------------------------------------|
| **secret**                | ✅        |                                                                  |
| **downwardAPI**           | ⚠️         | `metadata.namespace` only.                                       |
| **configMap**             | ✅        |                                                                  |
| **serviceAccountToken**   | ⚠️         | Supported without rotation. Expires in 3607 seconds by default.  |
| **clusterTrustBundle**    | ❌        |                                                                  |

## Usage Guide

To start using the project in your Kubernetes cluster, build and launch the project on your macOS host using the flags and environment variables described below. Don't forget to sign your binary with a necessary Virtualization.framework entitlements:

```shell
codesign --entitlements resources/vz.entitlements -s - <YOUR BINARY PATH>
```

If you need bridged networking however, vmnet entitlements are required (see `resources/release.entitlement`). Follow the steps in the [Networking](#networking) section on how to request them. After that you can edit Makefile with your configuration and use it to build and sign your custom binary.

See [Setup Workflow](#setup-workflow) on interacting with the project after it successfully connected to the cluster.

### Flags

All flags listed below are optional.

| Flag                                              | Type      | Default                           | Description                                                                                           |
|---------------------------------------------------|-----------|-----------------------------------|-------------------------------------------------------------------------------------------------------|
| `--nodename`                                      | String    | node hostname                     | The node's name as it will appear in the Kubernetes cluster.                                          |
| `--startup-timeout`                               | Integer   | `0`                               | The time in seconds to wait for the virtual kubelet to start.                                         |
| `--disable-taint`                                 | Bool      | `false`                           | Disables the taint that the virtual kubelet adds to the node.                                         |
| `--log-level`                                     | String    | `info`                            | The log level for the virtual kubelet.                                                                |
| `--pod-sync-workers`                              | Integer   | `10`                              | The number of workers to use for pod synchronization.                                                 |
| `--full-resync-period`                            | Integer   | `60`                              | The time in seconds between the node's full resyncs.                                                  |
| `--client-verify-ca`                              | String    | `APISERVER_CA_CERT_LOCATION` env  | The path to a CA certificate file to use to verify the Kubernetes API server's serving certificate.   |
| `--no-verify-clients`                             | Bool      | `false`                           | Turns off client verification of the Kubernetes API server's serving certificate.                     |
| `--authentication-token-webhook`                  | Bool      | `false`                           | Whether to use the TokenReview API to determine authentication for bearer tokens.                     |
| `--authentication-token-webhook-cache-ttl`        | Integer   | `0`                               | The duration to cache the authentication token webhook response.                                      |
| `--authorization-webhook-cache-authorized-ttl`    | Integer   | `0`                               | The duration to cache the authorization webhook response for authorized requests.                     |
| `--authorization-webhook-cache-unauthorized-ttl`  | Integer   | `0`                               | The duration to cache the authorization webhook response for unauthorized requests.                   |
| `--trace-sample-rate`                             | String    | Always Sample                     | The rate at which to sample traces.                                                                   |

### Environment Variables

| Environment Variable          | Required | Default                        | Description                                                                                                  |
|-------------------------------|:--------:|--------------------------------|--------------------------------------------------------------------------------------------------------------|
| `KUBECONFIG`                  |          | `~/.kube/config`               | The path to the kubeconfig file to use.                                                                      |
| `APISERVER_CERT_LOCATION`     | ✓        |                                | The path to the certificate file to use to authenticate to the Kubernetes API server.                        |
| `APISERVER_KEY_LOCATION`      | ✓        |                                | The path to the key file to use to authenticate to the Kubernetes API server.                                |
| `APISERVER_CA_CERT_LOCATION`  |          |                                | The path to a CA certificate file to verify the Kubernetes API server's serving certificate.                 |
| `OTEL_EXPORTER_OTLP_ENDPOINT` |          | Tracing disabled               | The endpoint to send trace data to.                                                                          |
| `OTEL_EXPORTER_OTLP_INSECURE` |          | `false`                        | Whether to disable TLS verification when sending trace data.                                                 |
| `OTEL_SERVICE_NAME`           |          |                                | The name of the service to use when sending trace data.                                                      |
| `VKUBELET_POD_IP`             |          |                                | The IP address to use for the virtual kubelet pod. Optional settings for debugging purposes.                 |
| `VZ_BRIDGE_INTERFACE`         |          |                                | The name of the bridge interface to use for the macOS VMs. Requires VMNet and VM Networking capabilities.    |
| `VZ_SSH_USER`                 | ✓        |                                | The username used when the virtual kubelet attempts to connect to the macOS VM over SSH.                     |
| `VZ_SSH_PASSWORD`             | ✓        |                                | The password used when the virtual kubelet attempts to connect to the macOS VM over SSH.                     |
| `DOCKER_HOST`                 |          | `unix:///var/run/docker.sock`  | The address of the Docker daemon to use for regular container support.                                       |

### Setup Workflow

1. **Create a macOS VM Image**

   Use tools compliant with Apple's Virtualization framework (e.g., [macosvm](https://github.com/s-u/macosvm)) to create a base macOS VM image.

1. **Package the Image**

   Package the VM image into the custom OCI format and push it to the registry using our fork of oras [oras-macos-vz](https://github.com/agoda-com/oras-macos-vz).

   ```shell
   oras-macos-vz push -h
   ```

1. **Prepare Kubernetes Pod Manifest**

   Write a Kubernetes Pod manifest that references the OCI image. See the [examples](example) folder for a sample manifest.

1. **Run the Pod**

   Virtual Kubelet will pick up your workload, download the image, and start the macOS VM. Status is reported back to Kubernetes as if it’s a regular container.

1. **Interact with the Pod**

   Use kubectl exec to interact with the macOS VM or other containers in the Pod.

### Local cache

The cache directory will be `~/Library/Caches/com.thesis.virtualization`. Cache includes OCI images and their digest files and pod mount volumes if you use empty_dir volumes.

## Example Workloads

Check the [examples](example) folder for:

- Sample Pod manifests.

- Example workloads, including hybrid Pods with macOS VMs and Docker side-cars.

## Roadmap

There are currently some features that we need to work on for our open-source release:

- Higher test coverage including open sourcing end to end testing approach.

- Have CI/CD pipeline for the open-source project to allow for a easier contribution.

## Community

If you have any questions or want to contribute to the project, feel free to reach out to us on the [GitHub Issues](https://github.com/agoda-com/macOS-vz-kubelet/issues) page.

### Related projects

The following projects are the bare-bone of macOS-vz-kubelet existence. They are heavily used by this project and are the foundation of the whole idea.

- [Virtual Kubelet](https://github.com/virtual-kubelet/virtual-kubelet) - Virtual Kubelet is an open source Kubernetes kubelet implementation.

- [Code-Hex/vz](https://github.com/Code-Hex/vz) - Create virtual machines and run Linux-based operating systems in Go using Apple Virtualization.framework.

- [oras-project/oras-go](https://github.com/oras-project/oras-go) - ORAS Go library for utilizing the Open Container Initiative (OCI).

## License

This project is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for more information.
