package nfsmount

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/virtual-kubelet/virtual-kubelet/log"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Mount prepares the network policy and mounts the NFS volume
func Mount(ctx context.Context, k8sClient kubernetes.Interface, namespace, serviceName, netpolName, macIP, localPath string) error {
	logger := log.G(ctx).WithFields(log.Fields{
		"service": serviceName,
		"netpol":  netpolName,
		"macIP":   macIP,
		"mount":   localPath,
	})

	if k8sClient == nil {
		return fmt.Errorf("kubernetes client is required for NFS mounts")
	}

	// 1. Patch the NetworkPolicy to allow this Mac's IP
	if netpolName != "" {
		logger.Infof("Patching NFS NetworkPolicy %s to allow Mac IP %s", netpolName, macIP)
		if err := patchNetworkPolicy(ctx, k8sClient, namespace, netpolName, macIP, true); err != nil {
			return fmt.Errorf("failed to patch NetworkPolicy: %w", err)
		}
	}

	// 2. Look up the Service to get the dynamically assigned NodePort
	logger.Infof("Looking up NFS Service %s for NodePort", serviceName)
	svc, err := k8sClient.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Service %s: %w", serviceName, err)
	}

	var nodePort int32
	for _, port := range svc.Spec.Ports {
		if port.Port == 2049 || port.Name == "nfs" {
			nodePort = port.NodePort
			break
		}
	}

	// 3. Get the Node Name where the Pod is running
	targetNodeName, err := waitForEndpoints(ctx, k8sClient, namespace, serviceName)
	if err != nil {
		return fmt.Errorf("timeout waiting for NFS endpoints: %w", err)
	}

	// 4. Look up the physical InternalIP of that specific Node
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

	// 5. Create local directory
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create mount path %s: %w", localPath, err)
	}

	// 6. Mount using the physical Node IP and the NodePort
	logger.Infof("Executing NodePort mount: targetIP=%s, nodePort=%d", targetIP, nodePort)
	cmdStr := fmt.Sprintf("vers=4,port=%d,rw", nodePort)
	targetStr := fmt.Sprintf("%s:%s", targetIP, localPath)
	cmd := exec.CommandContext(ctx, "mount", "-t", "nfs", "-o", cmdStr, targetStr, localPath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount command failed: %w (output: %s)", err, out)
	}
	// // 2. Look up the Service to get the ClusterIP
	// // logger.Infof("Looking up NFS Service %s", serviceName)
	// // svc, err := k8sClient.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	// // if err != nil {
	// // 	// Attempt rollback of netpol if we fail here
	// // 	_ = Unmount(ctx, k8sClient, namespace, netpolName, "")
	// // 	return fmt.Errorf("failed to get Service %s: %w", serviceName, err)
	// // }

	// // clusterIP := svc.Spec.ClusterIP
	// // if clusterIP == "" || clusterIP == "None" {
	// // 	_ = Unmount(ctx, k8sClient, namespace, netpolName, "")
	// // 	return fmt.Errorf("service %s has no ClusterIP", serviceName)
	// // }

	// // 3. Poll waiting for the endpoints to be non-empty (NFS pod is Ready)
	// logger.Infof("Waiting for endpoints to be ready for Service %s", serviceName)
	// targetIP, err := waitForEndpoints(ctx, k8sClient, namespace, serviceName)
	// if err != nil {
	// 	_ = Unmount(ctx, k8sClient, namespace, netpolName, "")
	// 	return fmt.Errorf("timeout waiting for NFS endpoints: %w", err)
	// }

	// // 4. Create the local mount directory
	// logger.Infof("Creating local mount directory %s", localPath)
	// if err := os.MkdirAll(localPath, 0755); err != nil {
	// 	_ = Unmount(ctx, k8sClient, namespace, netpolName, "")
	// 	return fmt.Errorf("failed to create mount path %s: %w", localPath, err)
	// }

	// /// 5. Run the macOS native mount command
	// // Note: Force NFSv4, target port 2049, and use the explicit export path
	// logger.Infof("Executing mount command: mount -t nfs -o vers=4,port=2049,rw %s:%s %s", targetIP, localPath, localPath)
	// cmd := exec.CommandContext(ctx, "mount", "-t", "nfs", "-o", "vers=4,port=2049,rw", fmt.Sprintf("%s:%s", targetIP, localPath), localPath)
	// out, err := cmd.CombinedOutput()
	// if err != nil {
	// 	_ = Unmount(ctx, k8sClient, namespace, netpolName, "")
	// 	return fmt.Errorf("mount command failed: %w (output: %s)", err, out)
	// }

	logger.Infof("Successfully mounted NFS at %s", localPath)
	return nil

}

// Unmount unmounts the local path and restores the NetworkPolicy
func Unmount(ctx context.Context, k8sClient kubernetes.Interface, namespace, netpolName, localPath string) error {
	logger := log.G(ctx)
	var finalErr error

	// 1. Unmount the path if provided
	if localPath != "" {
		logger.Infof("Unmounting NFS path %s", localPath)
		cmd := exec.CommandContext(ctx, "umount", localPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			logger.WithError(err).Warnf("umount command failed for %s: %s", localPath, out)
			finalErr = fmt.Errorf("umount failed: %w", err)
		} else {
			// clean up empty directory
			_ = os.Remove(localPath)
		}
	}

	// 2. Remove the IP from the network policy (restore deny-all)
	// Even if unmount failed, we should still try to secure the network
	if netpolName != "" && k8sClient != nil {
		logger.Infof("Restoring NFS NetworkPolicy %s to deny-all", netpolName)
		if err := patchNetworkPolicy(ctx, k8sClient, namespace, netpolName, "", false); err != nil {
			logger.WithError(err).Warnf("Failed to restore NetworkPolicy %s", netpolName)
			if finalErr == nil {
				finalErr = fmt.Errorf("failed to restore network policy: %w", err)
			}
		}
	}

	return finalErr
}

// patchNetworkPolicy updates the named NetworkPolicy to either allow or deny the given Mac IP
func patchNetworkPolicy(ctx context.Context, k8sClient kubernetes.Interface, namespace, name, macIP string, allow bool) error {
	netpol, err := k8sClient.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// For simplicity, we assume there is exactly one Ingress rule block, or create one.
	// Production deployments might have more complex policies.

	if allow {
		// Define the ingress rule for the Mac IP
		ipBlock := netv1.IPBlock{
			CIDR: fmt.Sprintf("%s/32", macIP),
		}

		rule := netv1.NetworkPolicyIngressRule{
			From: []netv1.NetworkPolicyPeer{
				{IPBlock: &ipBlock},
			},
		}

		if len(netpol.Spec.Ingress) == 0 {
			netpol.Spec.Ingress = []netv1.NetworkPolicyIngressRule{rule}
		} else {
			// Overwrite the first rule or append (we'll just overwrite for this bounded use case)
			netpol.Spec.Ingress[0] = rule
		}
	} else {
		// Deny all: Clear all ingress rules
		netpol.Spec.Ingress = []netv1.NetworkPolicyIngressRule{}
	}

	_, err = k8sClient.NetworkingV1().NetworkPolicies(namespace).Update(ctx, netpol, metav1.UpdateOptions{})
	return err
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
					// if ep.Conditions.Ready != nil && *ep.Conditions.Ready {
					// 	return nil
					// }
				}
			}
		}
	}
}
