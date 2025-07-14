package io

import (
	"bufio"
	"io"
)

// FlushingWriter wraps an io.Writer and automatically flushes after each write.
// This ensures real-time output for streaming operations like hook execution.
type FlushingWriter struct {
	w       io.Writer
	flusher interface{ Flush() error }
}

// NewFlushingWriter creates a new FlushingWriter. If the writer already supports
// flushing, it uses that directly. Otherwise, it wraps it in a bufio.Writer.
func NewFlushingWriter(w io.Writer) *FlushingWriter {
	fw := &FlushingWriter{w: w}

	// Check if writer already supports flushing
	if f, ok := w.(interface{ Flush() error }); ok {
		fw.flusher = f
	} else {
		// Wrap in bufio.Writer which supports flushing
		bw := bufio.NewWriter(w)
		fw.w = bw
		fw.flusher = bw
	}

	return fw
}

// Write writes data and immediately flushes to ensure real-time output.
func (fw *FlushingWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	if err != nil {
		return n, err
	}

	// Flush immediately after write for real-time output
	if fw.flusher != nil {
		if flushErr := fw.flusher.Flush(); flushErr != nil {
			return n, flushErr
		}
	}

	return n, nil
}

// Flush explicitly flushes any buffered data.
func (fw *FlushingWriter) Flush() error {
	if fw.flusher != nil {
		return fw.flusher.Flush()
	}
	return nil
}
