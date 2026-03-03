package provider

import (
	"context"
	"fmt"
	"io"

	"github.com/agoda-com/macOS-vz-kubelet/internal/node"
	"github.com/agoda-com/macOS-vz-kubelet/pkg/client"
	"github.com/agoda-com/macOS-vz-kubelet/pkg/event"

	// "github.com/agoda-com/macOS-vz-kubelet/pkg/metrics"

	dto "github.com/prometheus/client_model/go"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	"github.com/virtual-kubelet/virtual-kubelet/trace"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
)

const (
	ComponentName = "macos-exec-kubelet"
)

type MacOSExecProviderConfig struct {
	NodeName           string
	Platform           string
	InternalIP         string
	DaemonEndpointPort int32

	K8sClient     kubernetes.Interface
	EventRecorder event.EventRecorder
	PodsLister    corev1listers.PodLister
}

type MacOSExecProvider struct {
	runtimeClient client.RuntimeClient
	k8sClient     kubernetes.Interface
	podLister     corev1listers.PodLister

	eventRecorder event.EventRecorder

	nodeName           string
	platform           string
	internalIP         string
	daemonEndpointPort int32

	// *metrics.MacOSVZPodMetricsProvider
}

// NewMacOSExecProvider creates a new MacOSExec provider.
func NewMacOSExecProvider(ctx context.Context, runtimeClient client.RuntimeClient, config MacOSExecProviderConfig) (p *MacOSExecProvider, err error) {
	if config.Platform != "darwin" {
		return nil, errdefs.InvalidInputf("platform type %q is not supported", config.Platform)
	}

	p = &MacOSExecProvider{}
	p.runtimeClient = runtimeClient

	p.k8sClient = config.K8sClient
	p.podLister = config.PodsLister

	p.nodeName = config.NodeName
	p.platform = config.Platform
	p.internalIP = config.InternalIP
	p.daemonEndpointPort = config.DaemonEndpointPort

	p.eventRecorder = config.EventRecorder

	// TODO: Add metrics provider
	// p.MacOSVZPodMetricsProvider = metrics.NewMacOSVZPodMetricsProvider(p.nodeName, p.podLister, p.vzClient)
	return p, nil
}

var (
	errNotImplemented = fmt.Errorf("not implemented by MacOS provider")
)

// CreatePod takes a Kubernetes Pod and deploys it within the MacOS provider.
func (p *MacOSExecProvider) CreatePod(ctx context.Context, pod *corev1.Pod) (err error) {
	ctx = event.WithObjectRef(ctx, corev1.ObjectReference{
		Namespace: pod.Namespace,
		Name:      pod.Name,
		UID:       pod.UID,
	})
	ctx, span := trace.StartSpan(ctx, "MacOSExecProvider.CreatePod")
	defer func() {
		span.SetStatus(err)
		span.End()
	}()
	log.G(ctx).Debug("Received CreatePod request")

	configMaps, serviceAccountToken, err := p.extractPodCredentials(ctx, pod)
	if err != nil {
		return err
	}

	registryCreds, err := p.resolveImagePullCredentials(ctx, pod)
	if err != nil {
		p.eventRecorder.FailedToResolveImagePullSecrets(ctx, err)
		return errdefs.AsInvalidInput(err)
	}

	return p.runtimeClient.CreatePod(ctx, pod, serviceAccountToken, configMaps, registryCreds)
}

// UpdatePod takes a Kubernetes Pod and updates it within the provider.
func (p *MacOSExecProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) (err error) {
	ctx, span := trace.StartSpan(ctx, "MacOSExecProvider.UpdatePod")
	defer func() {
		span.SetStatus(err)
		span.End()
	}()
	log.G(ctx).Debug("Received UpdatePod request")
	return errNotImplemented
}

// DeletePod takes a Kubernetes Pod and deletes it from the provider.
func (p *MacOSExecProvider) DeletePod(ctx context.Context, pod *corev1.Pod) (err error) {
	ctx = event.WithObjectRef(ctx, corev1.ObjectReference{
		Namespace: pod.Namespace,
		Name:      pod.Name,
		UID:       pod.UID,
	})
	ctx, span := trace.StartSpan(ctx, "MacOSExecProvider.DeletePod")
	defer span.End()
	log.G(ctx).Debug("Received DeletePod request")

	// Execute delete request in go routine to avoid blocking the virtual kubelet thread
	go p.handleDeletePod(ctx, pod)

	return nil
}

func (p *MacOSExecProvider) handleDeletePod(ctx context.Context, pod *corev1.Pod) {
	var err error
	ctx, span := trace.StartSpan(ctx, "MacOSExecProvider.handleDeletePod")
	defer func() {
		span.SetStatus(err)
		span.End()
	}()

	gracePeriod := int64(0)
	if pod.DeletionGracePeriodSeconds != nil {
		gracePeriod = *pod.DeletionGracePeriodSeconds
	}

	// For process execution, we can support graceful shutdown if the process handles signals.
	err = p.runtimeClient.DeletePod(ctx, pod.Namespace, pod.Name, gracePeriod)
	if err != nil {
		log.G(ctx).WithError(err).Error("Failed to delete pod process")
		return
	}

	// If we successfully deleted the process, we can delete from K8s if needed.
	// Logic from original provider:
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: new(int64),
	}
	err = p.k8sClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, deleteOptions)
	if err != nil {
		log.G(ctx).WithError(err).Warn("Failed to delete pod from k8s")
	}
}

// GetPod retrieves a pod by name from the provider.
func (p *MacOSExecProvider) GetPod(ctx context.Context, namespace, name string) (pod *corev1.Pod, err error) {
	ctx, span := trace.StartSpan(ctx, "MacOSExecProvider.GetPod")
	defer func() {
		span.SetStatus(err)
		span.End()
	}()
	log.G(ctx).Debug("Received GetPod request")

	process, err := p.runtimeClient.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	if process.Pod != nil {
		// Return the pod spec we stored
		pod = process.Pod.DeepCopy()

		// Update status
		pod.Status = p.getPodStatusFromProcess(process)
		return pod, nil
	}

	return nil, errdefs.NotFound("pod spec not found in process")
}

// GetPodStatus retrieves the status of a pod by name from the provider.
func (p *MacOSExecProvider) GetPodStatus(ctx context.Context, namespace, name string) (ps *corev1.PodStatus, err error) {
	ctx, span := trace.StartSpan(ctx, "MacOSExecProvider.GetPodStatus")
	defer func() {
		span.SetStatus(err)
		span.End()
	}()
	log.G(ctx).Debug("Received GetPodStatus request")

	process, err := p.runtimeClient.GetPod(ctx, namespace, name)
	if err != nil {
		// If GetPod returns NotFound the process was already cleaned up — the
		// caller will treat the absence as the pod being gone.
		return nil, err
	}

	status := p.getPodStatusFromProcess(process)

	// Proactive cleanup: when the process has reached a terminal state the
	// process map entry has already been removed by GetPod above. Kick off an
	// async delete of the Kubernetes pod object so the controller can act on
	// the terminal status without waiting for VK's next polling cycle.
	if status.Phase == corev1.PodSucceeded || status.Phase == corev1.PodFailed {
		go func() {
			deleteOptions := metav1.DeleteOptions{GracePeriodSeconds: new(int64)}
			if delErr := p.k8sClient.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions); delErr != nil {
				// Pod may already be gone — log at Debug to avoid noise.
				log.G(ctx).WithError(delErr).Debug("Proactive pod K8s delete on terminal phase (may already be gone)")
			}
		}()
	}

	return &status, nil
}

func (p *MacOSExecProvider) getPodStatusFromProcess(process *client.PodProcess) corev1.PodStatus {
	// Simple mapping
	phase := corev1.PodRunning
	if process.ExitCode >= 0 {
		if process.ExitCode == 0 {
			phase = corev1.PodSucceeded
		} else {
			phase = corev1.PodFailed
		}
	} else if !process.StartedAt.IsZero() {
		phase = corev1.PodRunning
	} else {
		// Should not happen if Created
		phase = corev1.PodPending
	}

	// Containers status
	// For now assume single container
	containerName := "main" // fallback
	if process.Pod != nil && len(process.Pod.Spec.Containers) > 0 {
		containerName = process.Pod.Spec.Containers[0].Name
	}

	state := corev1.ContainerState{}
	if phase == corev1.PodRunning {
		state.Running = &corev1.ContainerStateRunning{
			StartedAt: metav1.NewTime(process.StartedAt),
		}
	} else {
		state.Terminated = &corev1.ContainerStateTerminated{
			ExitCode:   int32(process.ExitCode),
			StartedAt:  metav1.NewTime(process.StartedAt),
			FinishedAt: metav1.Now(), // approximate
		}
	}

	return corev1.PodStatus{
		Phase:     phase,
		HostIP:    p.internalIP,
		PodIP:     p.internalIP, // Exec process uses host network usually
		StartTime: &metav1.Time{Time: process.StartedAt},
		ContainerStatuses: []corev1.ContainerStatus{
			{
				Name:         containerName,
				State:        state,
				Ready:        phase == corev1.PodRunning,
				RestartCount: 0,
				Image:        "host-process",
				ImageID:      "",
			},
		},
	}
}

// GetPods retrieves a list of all pods running on the provider.
func (p *MacOSExecProvider) GetPods(ctx context.Context) (pods []*corev1.Pod, err error) {
	ctx, span := trace.StartSpan(ctx, "MacOSExecProvider.GetPods")
	defer func() {
		span.SetStatus(err)
		span.End()
	}()
	log.G(ctx).Debug("Received GetPods request")

	processes, err := p.runtimeClient.GetPodList(ctx)
	if err != nil {
		return nil, err
	}

	for _, process := range processes {
		if process.Pod != nil {
			pod := process.Pod.DeepCopy()
			pod.Status = p.getPodStatusFromProcess(process)
			pods = append(pods, pod)
		}
	}
	return pods, nil
}

// GetContainerLogs retrieves the logs of a container by name from the provider.
func (p *MacOSExecProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (in io.ReadCloser, err error) {
	return p.runtimeClient.GetContainerLogs(ctx, namespace, podName, containerName, opts)
}

// RunInContainer executes a command in a container in the pod.
func (p *MacOSExecProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) (err error) {
	ctx, span := trace.StartSpan(ctx, "MacOSExecProvider.RunInContainer")
	ctx = span.WithFields(ctx, log.Fields{
		"namespace":     namespace,
		"podName":       podName,
		"containerName": containerName,
		"cmd":           cmd,
	})
	defer func() {
		span.SetStatus(err)
		span.End()
	}()
	log.G(ctx).Debug("Received RunInContainer request")

	// Not truly supported by exec provider yet unless we implement attaching to existing process
	// or spawning a sibling process.
	// For "exec" provider, we might just spawn a new process on host?
	// But `ExecuteContainerCommand` in client might just spawn new process.
	// Let's defer to client.
	return p.runtimeClient.ExecuteContainerCommand(ctx, namespace, podName, containerName, cmd, attach)
}

// AttachToContainer attaches to the executing process of a container in the pod.
func (p *MacOSExecProvider) AttachToContainer(ctx context.Context, namespace, podName, containerName string, attach api.AttachIO) (err error) {
	return errNotImplemented
}

// PortForward forwards a local port to a port on the pod
func (p *MacOSExecProvider) PortForward(ctx context.Context, namespace, pod string, port int32, stream io.ReadWriteCloser) (err error) {
	return errNotImplemented
}

// GetMetricsResource satisfies the nodeutil.Provider interface.
// This provider does not expose Prometheus metrics; returning nil is valid.
func (p *MacOSExecProvider) GetMetricsResource(ctx context.Context) ([]*dto.MetricFamily, error) {
	return nil, nil
}

// GetStatsSummary satisfies the nodeutil.Provider interface.
// Process-based execution does not expose container-level resource stats.
func (p *MacOSExecProvider) GetStatsSummary(ctx context.Context) (*stats.Summary, error) {
	return &stats.Summary{
		Node: stats.NodeStats{
			NodeName: p.nodeName,
		},
	}, nil
}

// handlePreStopHooks is removed/simplified in this version as we don't have DiscardingExecIO or complex container logic yet.
// If needed, we can re-implement it using RuntimeClient.Example:
func (p *MacOSExecProvider) handlePreStopHooks(ctx context.Context, pod *corev1.Pod, gracePeriod int64) (err error) {
	// Basic implementation:
	discardingExec := node.DiscardingExecIO()
	for _, container := range pod.Spec.Containers {
		if lifecycle := container.Lifecycle; lifecycle != nil && lifecycle.PreStop != nil && lifecycle.PreStop.Exec != nil {
			containerName := container.Name
			command := lifecycle.PreStop.Exec.Command
			if err := p.runtimeClient.ExecuteContainerCommand(ctx, pod.Namespace, pod.Name, containerName, command, discardingExec); err != nil {
				// p.eventRecorder.FailedPreStopHook(ctx, containerName, command, err)
				return err
			}
		}
	}
	return nil
}
