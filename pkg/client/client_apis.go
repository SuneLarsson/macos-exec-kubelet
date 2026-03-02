package client

import (
	"context"
	"io"
	"time"

	"github.com/agoda-com/macOS-vz-kubelet/pkg/resource"

	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ProcessClientConfig holds configuration for creating a ProcessClient.
type ProcessClientConfig struct {
	// LogsDir is where stdout/stderr log files for each pod are written.
	LogsDir string
	// RunnerUser is the macOS username to run job processes as.
	// When empty, processes run as the current user (the kubelet's own account).
	RunnerUser string
}

// PodProcess represents a running process group for a pod.
type PodProcess struct {
	Namespace string
	Name      string
	Pid       int
	Command   []string
	StartedAt time.Time
	Pod       *corev1.Pod
	ExitCode  int
	ExitError error
}

// RuntimeClient defines the methods for managing pod processes.
type RuntimeClient interface {
	CreatePod(ctx context.Context, pod *corev1.Pod, serviceAccountToken string, configMaps map[string]*corev1.ConfigMap, creds resource.RegistryCredentialStore) error
	DeletePod(ctx context.Context, namespace, name string, gracePeriod int64) error
	GetPod(ctx context.Context, namespace, name string) (*PodProcess, error)
	GetPodList(ctx context.Context) (map[types.NamespacedName]*PodProcess, error)
	GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error)
	ExecuteContainerCommand(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error
	// AttachToContainer(ctx context.Context, namespace, podName, containerName string, attach api.AttachIO) error // Maybe support later
	GetPodStats(ctx context.Context, namespace, name string) ([]stats.ContainerStats, error)
}
