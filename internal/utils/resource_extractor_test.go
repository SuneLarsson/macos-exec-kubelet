package utils_test

import (
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestExtractCPURequest(t *testing.T) {
	tests := []struct {
		name         string
		resourceList corev1.ResourceList
		expectedCPU  uint
		expectError  bool
	}{
		{
			name: "Valid CPU request",
			resourceList: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("2"),
			},
			expectedCPU: 2,
			expectError: false,
		},
		{
			name: "Invalid CPU request",
			resourceList: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("2.74"),
			},
			expectedCPU: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, err := utils.ExtractCPURequest(tt.resourceList)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCPU, cpu)
			}
		})
	}
}

func TestExtractMemoryRequest(t *testing.T) {
	tests := []struct {
		name           string
		resourceList   corev1.ResourceList
		expectedMemory uint64
		expectError    bool
	}{
		{
			name: "Valid memory request",
			resourceList: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
			expectedMemory: 4 * 1024 * 1024 * 1024, // 4Gi in bytes
			expectError:    false,
		},
		{
			name: "Invalid memory request",
			resourceList: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("7.44Gi"),
			},
			expectedMemory: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory, err := utils.ExtractMemoryRequest(tt.resourceList)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMemory, memory)
			}
		})
	}
}
