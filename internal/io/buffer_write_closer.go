package io

import "bytes"

type BufferWriteCloser struct {
	buf *bytes.Buffer
}

func NewBufferWriteCloser(buf *bytes.Buffer) *BufferWriteCloser {
	return &BufferWriteCloser{
		buf: buf,
	}
}

func (wc *BufferWriteCloser) Write(p []byte) (n int, err error) {
	return wc.buf.Write(p)
}

func (wc *BufferWriteCloser) Close() error {
	// No actual close operation needed for in-memory buffer
	return nil
}
