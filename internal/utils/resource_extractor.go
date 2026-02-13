package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func ExtractCPURequest(rl corev1.ResourceList) (uint, error) {
	cpuSpec := rl[corev1.ResourceCPU]
	cpu, ok := cpuSpec.AsInt64()
	if !ok {
		return 0, fmt.Errorf("failed to parse CPU request")
	}

	return uint(cpu), nil
}

func ExtractMemoryRequest(rl corev1.ResourceList) (uint64, error) {
	memorySpec := rl[corev1.ResourceMemory]
	memory, ok := memorySpec.AsInt64()
	if !ok {
		return 0, fmt.Errorf("failed to parse memory request")
	}

	return uint64(memory), nil
}
