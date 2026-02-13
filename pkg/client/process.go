package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/agoda-com/macOS-vz-kubelet/pkg/resource"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

type ProcessClient struct {
	// Map of namespacedPodName -> PodProcess
	processes sync.Map
	logsDir   string
}

type processState struct {
	cmd        *exec.Cmd
	cancel     context.CancelFunc
	exitError  error
	waitDone   chan struct{}
	pid        int
	startedAt  time.Time
	podProcess *PodProcess
}

func NewProcessClient(logsDir string) *ProcessClient {
	return &ProcessClient{
		logsDir: logsDir,
	}
}

func (c *ProcessClient) CreatePod(ctx context.Context, pod *corev1.Pod, serviceAccountToken string, configMaps map[string]*corev1.ConfigMap, creds resource.RegistryCredentialStore) error {
	logger := log.G(ctx)

	if len(pod.Spec.Containers) == 0 {
		return fmt.Errorf("no containers in pod spec")
	}

	// For now, we only support the first container for simplicity in this iteration.
	// Expanding to multiple containers would require managing a group of processes.
	container := pod.Spec.Containers[0]

	key := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	if _, loaded := c.processes.Load(key); loaded {
		return fmt.Errorf("pod process already exists")
	}

	cmdParts := container.Command
	cmdParts = append(cmdParts, container.Args...)

	if len(cmdParts) == 0 {
		return fmt.Errorf("no command specified for container %s", container.Name)
	}

	command := cmdParts[0]
	args := cmdParts[1:]

	// Create context for the process
	pCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(pCtx, command, args...)

	// Setup environment
	cmd.Env = os.Environ()
	for _, env := range container.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	// Prepare logs
	podLogDir := filepath.Join(c.logsDir, pod.Namespace, pod.Name, container.Name)
	if err := os.MkdirAll(podLogDir, 0755); err != nil {
		cancel()
		return fmt.Errorf("failed to create log dir: %w", err)
	}

	stdoutFile, err := os.Create(filepath.Join(podLogDir, "stdout.log"))
	if err != nil {
		cancel()
		return err
	}
	stderrFile, err := os.Create(filepath.Join(podLogDir, "stderr.log"))
	if err != nil {
		stdoutFile.Close()
		cancel()
		return err
	}

	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	logger.Infof("Starting process for pod %s/%s: %s %v", pod.Namespace, pod.Name, command, args)

	if err := cmd.Start(); err != nil {
		logger.WithError(err).Error("Failed to start process")
		cancel()
		stdoutFile.Close()
		stderrFile.Close()
		return err
	}

	state := &processState{
		cmd:       cmd,
		cancel:    cancel,
		pid:       cmd.Process.Pid,
		startedAt: time.Now(),
		waitDone:  make(chan struct{}),
	}

	// Wait for process in background
	go func() {
		defer func() {
			stdoutFile.Close()
			stderrFile.Close()
			close(state.waitDone)
		}()
		state.exitError = cmd.Wait()
		if state.exitError != nil {
			logger.WithError(state.exitError).Infof("Process for pod %s/%s exited with error", pod.Namespace, pod.Name)
		} else {
			logger.Infof("Process for pod %s/%s exited successfully", pod.Namespace, pod.Name)
		}
	}()

	podProcess := &PodProcess{
		Namespace: pod.Namespace,
		Name:      pod.Name,
		Pid:       cmd.Process.Pid,
		Command:   cmdParts,
		StartedAt: state.startedAt,
		Pod:       pod,
	}

	state.podProcess = podProcess // Store it for retrieval

	// We store the processState pointer implicitly by associating it with the key in a separate map if needed,
	// but here we need to retrieve it for Kill.
	// Let's modify PodProcess or wrap it.
	// To keep checking strict matching, I will store a custom struct in the sync.Map
	// which contains both PodProcess data and the internal state.

	c.processes.Store(key, state)
	return nil
}

func (c *ProcessClient) DeletePod(ctx context.Context, namespace, name string, gracePeriod int64) error {
	key := types.NamespacedName{Namespace: namespace, Name: name}
	val, ok := c.processes.Load(key)
	if !ok {
		return errdefs.NotFound("pod process not found")
	}
	state := val.(*processState)

	// Cancel context to trigger Kill (via CommandContext) or manual signal
	// os/exec CommandContext sends SIGKILL when context is done, but we might want SIGTERM first.

	if state.cmd.Process != nil {
		state.cmd.Process.Signal(syscall.SIGTERM)
	}

	select {
	case <-state.waitDone:
		// Process exited
	case <-time.After(time.Duration(gracePeriod) * time.Second):
		// Force kill
		state.cancel() // This kills the process
		<-state.waitDone
	}

	c.processes.Delete(key)
	return nil
}

func (c *ProcessClient) GetPod(ctx context.Context, namespace, name string) (*PodProcess, error) {
	key := types.NamespacedName{Namespace: namespace, Name: name}
	val, ok := c.processes.Load(key)
	if !ok {
		return nil, errdefs.NotFound("pod process not found")
	}
	state := val.(*processState)

	var exitCode int
	var exitErr error

	select {
	case <-state.waitDone:
		// Exited
		exitErr = state.exitError
		if exitErr != nil {
			if ee, ok := exitErr.(*exec.ExitError); ok {
				exitCode = ee.ExitCode()
			} else {
				exitCode = 1 // Basic error
			}
		}
	default:
		// Running
		exitCode = -1 // Running
	}

	// Update stored PodProcess in case we want to return the updated one
	// But PodProcess struct is a value in map? No, it's pointer.
	// However, state.podProcess is what we return.
	state.podProcess.ExitCode = exitCode
	state.podProcess.ExitError = exitErr

	return state.podProcess, nil
}

func (c *ProcessClient) GetPodList(ctx context.Context) (map[types.NamespacedName]*PodProcess, error) {
	result := make(map[types.NamespacedName]*PodProcess)
	c.processes.Range(func(key, value interface{}) bool {
		k := key.(types.NamespacedName)
		state := value.(*processState)
		result[k] = state.podProcess
		return true
	})
	return result, nil
}

func (c *ProcessClient) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	// Dumb implementation: just read the whole file or tail it.
	// For now, return the file opened.
	podLogDir := filepath.Join(c.logsDir, namespace, podName, containerName)
	// Try stdout or stderr

	// NOTE: This does not support follow/tail properly without more complex logic.
	path := filepath.Join(podLogDir, "stdout.log")
	return os.Open(path)
}

func (c *ProcessClient) ExecuteContainerCommand(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error {
	return fmt.Errorf("executing in running process not supported yet")
}

func (c *ProcessClient) GetPodStats(ctx context.Context, namespace, name string) ([]stats.ContainerStats, error) {
	return nil, nil // Not implemented
}
