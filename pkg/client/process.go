package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

const (
	// logRotateMaxBytes is the size threshold above which a log file is rotated
	// before being re-opened for a new pod run.
	logRotateMaxBytes = 50 * 1024 * 1024 // 50 MB

	// pidFileName is written next to stdout/stderr to enable orphan cleanup on restart.
	pidFileName = "pid"
)

type ProcessClient struct {
	// Map of namespacedPodName -> PodProcess
	processes  sync.Map
	logsDir    string
	credential *resolvedCredential // nil means: run as current user
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

// NewProcessClient creates a ProcessClient. cfg.RunnerUser is resolved to a
// UID/GID at construction time so a bad username causes an immediate error
// rather than a per-job failure.
//
// On startup, NewProcessClient scans logsDir for PID files left by a previous
// kubelet instance and kills any surviving orphan processes before accepting
// new work.
func NewProcessClient(cfg ProcessClientConfig) (*ProcessClient, error) {
	cred, err := lookupRunnerCredential(cfg.RunnerUser)
	if err != nil {
		return nil, err
	}
	c := &ProcessClient{
		logsDir:    cfg.LogsDir,
		credential: cred,
	}
	if cfg.LogsDir != "" {
		c.scanAndKillOrphans(cfg.LogsDir)
	}
	return c, nil
}

// scanAndKillOrphans walks logsDir looking for pid files written by a previous
// kubelet run. Any process that is still alive is SIGKILL-ed and the log
// directory is removed so the next run starts clean.
func (c *ProcessClient) scanAndKillOrphans(logsDir string) {
	ctx := context.Background()
	logger := log.G(ctx)

	// Walk: logsDir/<namespace>/<podName>/<containerName>/pid
	_ = filepath.Walk(logsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || info.Name() != pidFileName {
			return nil
		}

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			logger.WithError(readErr).Warnf("orphan scan: cannot read pid file %s", path)
			return nil
		}

		pidStr := strings.TrimSpace(string(raw))
		// Format: "<pid> <startEpochNano>"
		parts := strings.SplitN(pidStr, " ", 2)
		pid, convErr := strconv.Atoi(parts[0])
		if convErr != nil || pid <= 0 {
			logger.Warnf("orphan scan: invalid pid in %s: %q", path, pidStr)
			return nil
		}

		var savedStartNano int64
		if len(parts) == 2 {
			savedStartNano, _ = strconv.ParseInt(parts[1], 10, 64)
		}

		// Check liveness with signal 0
		proc, findErr := os.FindProcess(pid)
		if findErr != nil {
			// Process gone — just clean up
			cleanupOrphanDir(logger, filepath.Dir(path))
			return nil
		}

		// signal 0 confirms whether the PID is alive (Unix only; on macOS this works)
		if sigErr := proc.Signal(syscall.Signal(0)); sigErr != nil {
			// ESRCH or permission error — either gone or not ours
			cleanupOrphanDir(logger, filepath.Dir(path))
			return nil
		}

		// Guard against PID reuse: compare process start time if we have it
		if savedStartNano != 0 {
			actualStartNano := processStartTimeNano(pid)
			if actualStartNano != 0 && actualStartNano != savedStartNano {
				logger.Infof("orphan scan: pid %d reused (saved start %d, actual %d) — skipping kill",
					pid, savedStartNano, actualStartNano)
				cleanupOrphanDir(logger, filepath.Dir(path))
				return nil
			}
		}

		// Kill the orphan
		logger.Infof("orphan scan: killing orphan process pid=%d from %s", pid, path)
		if killErr := proc.Signal(syscall.SIGKILL); killErr != nil {
			logger.WithError(killErr).Warnf("orphan scan: failed to kill pid %d", pid)
		}
		cleanupOrphanDir(logger, filepath.Dir(path))
		return nil
	})
}

func cleanupOrphanDir(logger log.Logger, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		logger.WithError(err).Warnf("orphan scan: failed to remove dir %s", dir)
	}
}

// rotateLogFile renames path → path+".1" if the file already exists and
// exceeds logRotateMaxBytes. This keeps unbounded log growth in check for
// long-lived pods.
func rotateLogFile(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil // nothing to rotate
	}
	if err != nil {
		return err
	}
	if info.Size() < logRotateMaxBytes {
		return nil
	}
	rotated := path + ".1"
	return os.Rename(path, rotated)
}

// writePidFile writes "<pid> <startEpochNano>\n" to the pid file in podLogDir.
func writePidFile(podLogDir string, pid int, startedAt time.Time) error {
	content := fmt.Sprintf("%d %d\n", pid, startedAt.UnixNano())
	return os.WriteFile(filepath.Join(podLogDir, pidFileName), []byte(content), 0644)
}

func (c *ProcessClient) CreatePod(ctx context.Context, pod *corev1.Pod, serviceAccountToken string, configMaps map[string]*corev1.ConfigMap, creds resource.RegistryCredentialStore) error {
	logger := log.G(ctx)

	if len(pod.Spec.Containers) == 0 {
		return fmt.Errorf("no containers in pod spec")
	}

	// Warn if the user defined multiple containers — only the first will run.
	if len(pod.Spec.Containers) > 1 {
		logger.Warn("Only first container is supported; additional containers will be ignored")
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

	// Apply working directory if specified
	if container.WorkingDir != "" {
		cmd.Dir = container.WorkingDir
	}

	// Drop to runner user if configured
	applyCredential(cmd, c.credential)

	// Setup environment
	cmd.Env = os.Environ()
	for _, env := range container.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	// Prepare logs directory; chown so the runner user can write to it
	podLogDir := filepath.Join(c.logsDir, pod.Namespace, pod.Name, container.Name)
	if err := os.MkdirAll(podLogDir, 0755); err != nil {
		cancel()
		return fmt.Errorf("failed to create log dir: %w", err)
	}
	if err := chownLogDir(podLogDir, c.credential); err != nil {
		cancel()
		return fmt.Errorf("failed to chown log dir: %w", err)
	}

	// Rotate existing log files before (re-)opening them
	stdoutPath := filepath.Join(podLogDir, "stdout.log")
	stderrPath := filepath.Join(podLogDir, "stderr.log")
	if err := rotateLogFile(stdoutPath); err != nil {
		cancel()
		return fmt.Errorf("failed to rotate stdout log: %w", err)
	}
	if err := rotateLogFile(stderrPath); err != nil {
		cancel()
		return fmt.Errorf("failed to rotate stderr log: %w", err)
	}

	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		cancel()
		return err
	}
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		stdoutFile.Close()
		cancel()
		return err
	}

	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	if c.credential != nil {
		logger.Infof("Starting process for pod %s/%s as UID=%d: %s %v", pod.Namespace, pod.Name, c.credential.uid, command, args)
	} else {
		logger.Infof("Starting process for pod %s/%s: %s %v", pod.Namespace, pod.Name, command, args)
	}

	if err := cmd.Start(); err != nil {
		logger.WithError(err).Error("Failed to start process")
		cancel()
		stdoutFile.Close()
		stderrFile.Close()
		return err
	}

	startedAt := time.Now()

	// Write PID file immediately so orphan cleanup can find it on the next restart
	if err := writePidFile(podLogDir, cmd.Process.Pid, startedAt); err != nil {
		logger.WithError(err).Warn("Failed to write pid file; orphan cleanup will not cover this pod")
	}

	state := &processState{
		cmd:       cmd,
		cancel:    cancel,
		pid:       cmd.Process.Pid,
		startedAt: startedAt,
		waitDone:  make(chan struct{}),
	}

	// Wait for process in background
	go func() {
		defer func() {
			stdoutFile.Close()
			stderrFile.Close()
			// Remove the pid file once the process has exited cleanly — it is
			// no longer an orphan candidate.
			_ = os.Remove(filepath.Join(podLogDir, pidFileName))
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
		StartedAt: startedAt,
		Pod:       pod,
	}

	state.podProcess = podProcess
	c.processes.Store(key, state)
	return nil
}

func (c *ProcessClient) DeletePod(ctx context.Context, namespace, name string, gracePeriod int64) error {
	// 1. Always ensure the log directory is cleaned up, regardless of whether
	// the process is still in memory. If the kubelet restarted and the process
	// previously exited, the map will be empty but logs will still be on disk.
	podLogDir := filepath.Join(c.logsDir, namespace, name)
	if err := os.RemoveAll(podLogDir); err != nil {
		log.G(ctx).WithError(err).Warn("Failed to remove pod log directory")
	}
	// Attempt to remove the namespace directory in case this was the last pod.
	// os.Remove will naturally fail (and we ignore the error) if it's not empty.
	_ = os.Remove(filepath.Dir(podLogDir))

	key := types.NamespacedName{Namespace: namespace, Name: name}
	val, ok := c.processes.Load(key)
	if !ok {
		return errdefs.NotFound("pod process not found in memory")
	}
	state := val.(*processState)

	// Send SIGTERM first; fall back to SIGKILL via context cancel after grace period
	if state.cmd.Process != nil {
		state.cmd.Process.Signal(syscall.SIGTERM)
	}

	select {
	case <-state.waitDone:
		// Process exited cleanly
	case <-time.After(time.Duration(gracePeriod) * time.Second):
		// Force kill
		state.cancel()
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
		// Exited — capture the result. We DO NOT delete from the map here so
		// that the pod stays in Succeeded/Failed state and logs remain
		// accessible until K8s explicitly calls DeletePod.
		exitErr = state.exitError
		if exitErr != nil {
			if ee, ok := exitErr.(*exec.ExitError); ok {
				exitCode = ee.ExitCode()
			} else {
				exitCode = 1
			}
		}
	default:
		// Running
		exitCode = -1
	}

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
	podLogDir := filepath.Join(c.logsDir, namespace, podName, containerName)
	path := filepath.Join(podLogDir, "stdout.log")
	return os.Open(path)
}

func (c *ProcessClient) ExecuteContainerCommand(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error {
	return fmt.Errorf("executing in running process not supported yet")
}

func (c *ProcessClient) GetPodStats(ctx context.Context, namespace, name string) ([]stats.ContainerStats, error) {
	return nil, nil // Not implemented
}
