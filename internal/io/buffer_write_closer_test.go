package io_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	localio "github.com/agoda-com/macOS-vz-kubelet/internal/io"
)

func TestBufferWriteCloser(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
		isEmpty  bool
	}{
		{
			name:    "write empty data",
			input:   []byte{},
			isEmpty: true,
		},
		{
			name:     "write simple string",
			input:    []byte("hello world"),
			expected: []byte("hello world"),
		},
		{
			name:     "write binary data",
			input:    []byte{0x00, 0x01, 0x02, 0x03},
			expected: []byte{0x00, 0x01, 0x02, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			buf := &bytes.Buffer{}
			wc := localio.NewBufferWriteCloser(buf)

			// Test Write
			n, err := wc.Write(tt.input)
			require.NoError(t, err)
			assert.Equal(t, len(tt.input), n)

			if tt.isEmpty {
				assert.Empty(t, buf.Bytes())
			} else {
				assert.Equal(t, tt.expected, buf.Bytes())
			}

			// Test Close
			err = wc.Close()
			assert.NoError(t, err)
		})
	}
}

func TestBufferWriteCloser_MultipleWrites(t *testing.T) {
	// Setup
	buf := &bytes.Buffer{}
	wc := localio.NewBufferWriteCloser(buf)

	// Test multiple writes
	writes := [][]byte{
		[]byte("first "),
		[]byte("second "),
		[]byte("third"),
	}
	expected := []byte("first second third")

	for _, data := range writes {
		n, err := wc.Write(data)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
	}

	assert.Equal(t, expected, buf.Bytes())
	assert.NoError(t, wc.Close())
}
