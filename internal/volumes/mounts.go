package volumes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/virtual-kubelet/virtual-kubelet/log"
	corev1 "k8s.io/api/core/v1"
)

const PodVolPerms os.FileMode = 0755

// Mount represents a universal mount point in a container.
// Note: This is a simplified version of the actual implementation
// and can be replaced by containerd's Mount type whenever (if) containerd
// is integrated into the project.
type Mount struct {
	Name          string
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

// CreateContainerMounts creates the mounts for a container based on the pod spec.
func CreateContainerMounts(ctx context.Context, podVolRoot string, container corev1.Container, pod *corev1.Pod, serviceAccountToken string, configMaps map[string]*corev1.ConfigMap) ([]Mount, error) {
	mounts := []Mount{}
	for _, mountSpec := range container.VolumeMounts {
		podVolSpec := findPodVolumeSpec(pod, mountSpec.Name)
		if podVolSpec == nil {
			log.G(ctx).Debugf("Container volume mount %s not found in Pod spec", mountSpec.Name)
			continue
		}

		// Common fields to all mount types
		newMount := Mount{
			Name:          mountSpec.Name,
			ContainerPath: filepath.Join(mountSpec.MountPath, mountSpec.SubPath),
			ReadOnly:      mountSpec.ReadOnly,
		}
		// Iterate over the volume types we care about
		if podVolSpec.HostPath != nil {
			// create the host path if it doesn't exist
			err := os.MkdirAll(podVolSpec.HostPath.Path, PodVolPerms)
			if err != nil {
				return nil, fmt.Errorf("error making hostPath for path %s: %w", podVolSpec.HostPath.Path, err)
			}
			newMount.HostPath = podVolSpec.HostPath.Path
		} else if podVolSpec.EmptyDir != nil {
			// TODO: Currently ignores the SizeLimit
			newMount.HostPath = filepath.Join(podVolRoot, mountSpec.Name)
			err := os.MkdirAll(newMount.HostPath, PodVolPerms)
			if err != nil {
				return nil, fmt.Errorf("error making emptyDir for path %s: %w", newMount.HostPath, err)
			}
		} else if podVolSpec.Projected != nil {
			newMount.HostPath = filepath.Join(podVolRoot, mountSpec.Name)
			err := os.MkdirAll(newMount.HostPath, PodVolPerms)
			if err != nil {
				return nil, fmt.Errorf("error making projected for path %s: %w", newMount.HostPath, err)
			}
			for _, source := range podVolSpec.Projected.Sources {
				if source.ServiceAccountToken != nil {
					err := os.WriteFile(filepath.Join(newMount.HostPath, source.ServiceAccountToken.Path), []byte(serviceAccountToken), PodVolPerms)
					if err != nil {
						return nil, fmt.Errorf("error writing service account token: %w", err)
					}
				}
				if source.ConfigMap != nil {
					configMap := configMaps[source.ConfigMap.Name]
					if configMap == nil {
						return nil, fmt.Errorf("config map %s not found", source.ConfigMap.Name)
					}

					for _, keyToPath := range source.ConfigMap.Items {
						value := configMap.Data[keyToPath.Key]
						mode := PodVolPerms
						if keyToPath.Mode != nil {
							mode = os.FileMode(*keyToPath.Mode)
						}
						err := os.WriteFile(filepath.Join(newMount.HostPath, keyToPath.Path), []byte(value), mode)
						if err != nil {
							return nil, fmt.Errorf("error writing config map: %w", err)
						}
					}
				}
				if source.DownwardAPI != nil {
					// currently only namespace is supported
					for _, item := range source.DownwardAPI.Items {
						if item.FieldRef.FieldPath == "metadata.namespace" {
							mode := PodVolPerms
							if item.Mode != nil {
								mode = os.FileMode(*item.Mode)
							}
							err := os.WriteFile(filepath.Join(newMount.HostPath, item.Path), []byte(pod.Namespace), mode)
							if err != nil {
								return nil, fmt.Errorf("error writing downward API: %w", err)
							}
						}
					}
				}
			}
		} else {
			continue
		}
		mounts = append(mounts, newMount)
	}

	return mounts, nil
}

// findPodVolumeSpec searches for a particular volume spec by name in the Pod spec
func findPodVolumeSpec(pod *corev1.Pod, name string) *corev1.VolumeSource {
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == name {
			return &volume.VolumeSource
		}
	}
	return nil
}
