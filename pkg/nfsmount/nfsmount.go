package nfsmount

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/virtual-kubelet/virtual-kubelet/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Mount prepares the network policy and mounts the NFS volume
func Mount(ctx context.Context, k8sClient kubernetes.Interface, namespace, serviceName, netpolName, macIP, localPath string) error {
	logger := log.G(ctx).WithFields(log.Fields{
		"service": serviceName,
		"macIP":   macIP,
		"mount":   localPath,
	})

	if k8sClient == nil {
		return fmt.Errorf("kubernetes client is required for NFS mounts")
	}

	logger.Infof("Waiting for endpoints to be ready for Service %s", serviceName)
	targetNodeName, err := waitForEndpoints(ctx, k8sClient, namespace, serviceName)
	if err != nil {
		return fmt.Errorf("timeout waiting for NFS endpoints: %w", err)
	}

	// Look up the physical InternalIP of that specific Node
	node, err := k8sClient.CoreV1().Nodes().Get(ctx, targetNodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Node %s: %w", targetNodeName, err)
	}

	var targetIP string
	for _, addr := range node.Status.Addresses {
		if addr.Type == "InternalIP" {
			targetIP = addr.Address
			break
		}
	}

	if targetIP == "" {
		return fmt.Errorf("could not find InternalIP for Node %s", targetNodeName)
	}

	// Explicitly define the remote export and the safe local macOS path
	remotePath := "/"
	safeLocalPath := "/tmp/sommarjobb-jupyterlaunch"

	// Create local directory in the writable /tmp space to bypass macOS SIP
	logger.Infof("Creating local safe mount directory %s", safeLocalPath)
	if err := os.MkdirAll(safeLocalPath, 0755); err != nil {
		return fmt.Errorf("failed to create mount path %s: %w", safeLocalPath, err)
	}

	// Mount using the physical Node IP, port 2049 (HostPort), and separated paths
	logger.Infof("Executing HostPort mount: targetIP=%s, port=2049", targetIP)
	cmdStr := "vers=4,port=2049,rw"
	targetStr := fmt.Sprintf("%s:%s", targetIP, remotePath)

	cmd := exec.CommandContext(ctx, "mount", "-t", "nfs", "-o", cmdStr, targetStr, safeLocalPath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount command failed: %w (output: %s)", err, out)
	}

	logger.Infof("Successfully mounted NFS share to %s", safeLocalPath)
	return nil
}

// Unmount unmounts the local path
func Unmount(ctx context.Context, k8sClient kubernetes.Interface, namespace, netpolName, localPath string) error {
	logger := log.G(ctx)

	path := localPath
	if path == "" {
		path = "/tmp/sommarjobb-jupyterlaunch"
	}

	logger.Infof("Unmounting NFS path %s", path)
	cmd := exec.CommandContext(ctx, "umount", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		logger.WithError(err).Warnf("umount command failed for %s: %s", path, out)
		// Try fallback if the formal localPath isn't mounted but safeLocalPath is
		if path != "/tmp/sommarjobb-jupyterlaunch" {
			fallback := "/tmp/sommarjobb-jupyterlaunch"
			logger.Infof("Trying fallback umount for %s", fallback)
			cmd = exec.CommandContext(ctx, "umount", fallback)
			if fallbackOut, fallbackErr := cmd.CombinedOutput(); fallbackErr == nil {
				_ = os.Remove(fallback)
				return nil
			} else {
				logger.WithError(fallbackErr).Warnf("fallback umount failed for %s: %s", fallback, fallbackOut)
			}
		}
		return fmt.Errorf("umount failed: %w", err)
	}

	// clean up empty directory
	_ = os.Remove(path)

	return nil
}

// waitForEndpoints blocks until the service has at least one ready endpoint
func waitForEndpoints(ctx context.Context, k8sClient kubernetes.Interface, namespace, serviceName string) (string, error) {
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("timed out waiting for endpoints for %s", serviceName)
		case <-ticker.C:
			// check endpoints
			epList, err := k8sClient.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", serviceName),
			})

			if err != nil {
				continue
			}

			// Check if we have at least one ready endpoint address
			for _, slice := range epList.Items {
				for _, ep := range slice.Endpoints {
					if ep.Conditions.Ready != nil && *ep.Conditions.Ready {
						if ep.NodeName != nil {
							return *ep.NodeName, nil
						}
					}
				}
			}
		}
	}
}
