package client

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agoda-com/macOS-vz-kubelet/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProcessClient_CreatePod(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vk-exec-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	c, err := NewProcessClient(ProcessClientConfig{LogsDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "test-container",
					Command: []string{"ping", "127.0.0.1", "-n", "2"}, // Windows ping
				},
			},
		},
	}

	// Adjust command for non-windows if needed, but user is on windows?
	// The user environment is Windows, but the TARGET is macOS-vz-kubelet which runs on macOS.
	// Wait, the user said "The USER's OS version is windows." BUT the project is "macos-exec-kubelet".
	// Code execution happens on the USER's machine if I run tests locally.
	// If I run tests on Windows, I should use Windows commands?
	// Or is the user developing on Windows for macOS? "macos-exec-kubelet" implies it runs on macOS.
	// If I run `go test`, it runs on Windows.
	// I should probably skip the test if not on Darwin? Or use cross-platform command.
	// "ping" availability on Windows/Mac differs slightly (args), but "echo" is shell builtin usually.
	// "go" binary is available. "go version".

	// I will use a simple command that works or just check if it fails gracefully.
	// Using "hostname" is usually safe.

	pod.Spec.Containers[0].Command = []string{"hostname"}

	ctx := context.Background()
	err = c.CreatePod(ctx, pod, "", nil, resource.RegistryCredentialStore{})
	if err != nil {
		t.Fatalf("CreatePod failed: %v", err)
	}

	// Check if process is running/exists
	p, err := c.GetPod(ctx, "default", "test-pod")
	if err != nil {
		t.Fatalf("GetPod failed: %v", err)
	}
	if p == nil {
		t.Fatal("GetPod returned nil")
	}
	if p.Pod.Name != "test-pod" {
		t.Errorf("Expected pod name test-pod, got %s", p.Pod.Name)
	}

	// Wait for it to finish
	time.Sleep(1 * time.Second)

	// Logs check
	logFile := filepath.Join(tmpDir, "default", "test-pod", "test-container", "stdout.log")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file not created: %s", logFile)
	}

	// Delete
	err = c.DeletePod(ctx, "default", "test-pod", 0)
	if err != nil {
		t.Errorf("DeletePod failed: %v", err)
	}
}
