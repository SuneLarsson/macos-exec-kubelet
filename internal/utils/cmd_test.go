package utils_test

import (
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/utils"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestBuildExportEnvCommand(t *testing.T) {
	tests := []struct {
		name     string
		env      corev1.EnvVar
		expected string
	}{
		{
			name: "Single line value",
			env: corev1.EnvVar{
				Name:  "SINGLE_LINE",
				Value: "simple_value",
			},
			expected: "export SINGLE_LINE=\"simple_value\"\n",
		},
		{
			name: "Single line value with special characters",
			env: corev1.EnvVar{
				Name:  "SPECIAL_CHARS",
				Value: "value_with_special_chars!@#$%^&*()",
			},
			expected: "export SPECIAL_CHARS=\"value_with_special_chars!@#$%^&*()\"\n",
		},
		{
			name: "Multi-line value",
			env: corev1.EnvVar{
				Name:  "MULTI_LINE",
				Value: "line1\nline2\nline3",
			},
			expected: "export MULTI_LINE=$(cat <<'ESCAPE_EOF'\nline1\nline2\nline3\nESCAPE_EOF\n)\n",
		},
		{
			name: "Empty value",
			env: corev1.EnvVar{
				Name:  "EMPTY_VALUE",
				Value: "",
			},
			expected: "export EMPTY_VALUE=\"\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.BuildExportEnvCommand(tt.env)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildExecCommandString(t *testing.T) {
	tests := []struct {
		name        string
		cmd         []string
		env         []corev1.EnvVar
		expected    string
		expectError bool
	}{
		{
			name: "Valid command with environment variables",
			cmd:  []string{"sh", "-c", "echo Hello"},
			env: []corev1.EnvVar{
				{Name: "FOO", Value: "bar"},
				{Name: "BAZ", Value: "qux"},
			},
			expected:    "export FOO=\"bar\"\nexport BAZ=\"qux\"\nsh -c $'echo Hello'",
			expectError: false,
		},
		{
			name:        "Invalid command (less than 3 elements)",
			cmd:         []string{"sh", "-c"},
			env:         []corev1.EnvVar{},
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid command (second element is not -c)",
			cmd:         []string{"sh", "-x", "echo Hello"},
			env:         []corev1.EnvVar{},
			expected:    "",
			expectError: true,
		},
		{
			name:        "Command with additional arguments",
			cmd:         []string{"sh", "-c", "echo Hello", "arg1", "arg2"},
			env:         []corev1.EnvVar{},
			expected:    "sh -c $'echo Hello' \"arg1\" \"arg2\"",
			expectError: false,
		},
		{
			name:        "Command with no additional arguments",
			cmd:         []string{"sh", "-c", "echo Hello"},
			env:         []corev1.EnvVar{},
			expected:    "sh -c $'echo Hello'",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.BuildExecCommandString(tt.cmd, tt.env)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
