package utils_test

import (
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeAppleCPUModelForK8sLabel(t *testing.T) {
	// Define the test cases
	tests := []struct {
		name           string // Name of the test case
		input          string // Input CPU model name
		expectedOutput string // Expected simplified model name
	}{
		{
			name:           "Basic Apple CPU model",
			input:          "Apple M1 Pro",
			expectedOutput: "M1_Pro",
		},
		{
			name:           "Apple CPU model with spaces",
			input:          "Apple A12 Bionic",
			expectedOutput: "A12_Bionic",
		},
		{
			name:           "Apple CPU model with no changes needed",
			input:          "M1 Pro",
			expectedOutput: "M1_Pro",
		},
		{
			name:           "Empty input",
			input:          "",
			expectedOutput: "",
		},
		{
			name:           "Apple CPU model with extra spaces",
			input:          "Apple M1   Max",
			expectedOutput: "M1_Max", // Adjusted expected output as per regex to handle extra spaces
		},
		{
			name:           "Non-Apple model (no changes)",
			input:          "Intel Core i9",
			expectedOutput: "Intel_Core_i9",
		},
		{
			name:           "Model with invalid characters",
			input:          "Apple M1!Pro",
			expectedOutput: "M1_Pro", // Invalid character '!' should be removed
		},
		{
			name:           "Model starting and ending with invalid characters",
			input:          "-Apple M1 Pro-",
			expectedOutput: "M1_Pro", // Hyphens at the beginning and end should be removed
		},
		{
			name:           "Model with multiple invalid characters",
			input:          "Apple M1@Pro#",
			expectedOutput: "M1_Pro", // '@' and '#' are invalid and should be removed
		},
		{
			name:           "Model with multiple invalid characters",
			input:          "Apple M2 Pro (Virtual)",
			expectedOutput: "M2_Pro_Virtual",
		},
	}

	// Loop over the test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function with the input
			output := utils.SanitizeAppleCPUModelForK8sLabel(tc.input)

			// Assert that the output matches the expected result
			assert.Equal(t, tc.expectedOutput, output)
		})
	}
}
