package io

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFlusher is a mock writer that tracks flush calls
type mockFlusher struct {
	bytes.Buffer
	flushCount int
	flushError error
}

func (m *mockFlusher) Flush() error {
	m.flushCount++
	return m.flushError
}

func TestNewFlushingWriter(t *testing.T) {
	t.Run("wraps non-flushing writer", func(t *testing.T) {
		var buf bytes.Buffer
		fw := NewFlushingWriter(&buf)

		assert.NotNil(t, fw)
		assert.NotNil(t, fw.flusher)
		// Original writer should be wrapped
		assert.NotEqual(t, &buf, fw.w)
	})

	t.Run("uses existing flusher", func(t *testing.T) {
		mf := &mockFlusher{}
		fw := NewFlushingWriter(mf)

		assert.NotNil(t, fw)
		assert.Equal(t, mf, fw.flusher)
		assert.Equal(t, mf, fw.w)
	})
}

func TestFlushingWriter_Write(t *testing.T) {
	t.Run("writes and flushes immediately", func(t *testing.T) {
		mf := &mockFlusher{}
		fw := NewFlushingWriter(mf)

		data := []byte("test data")
		n, err := fw.Write(data)

		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, "test data", mf.String())
		assert.Equal(t, 1, mf.flushCount, "should flush after write")
	})

	t.Run("multiple writes flush each time", func(t *testing.T) {
		mf := &mockFlusher{}
		fw := NewFlushingWriter(mf)

		_, err := fw.Write([]byte("first"))
		require.NoError(t, err)
		assert.Equal(t, 1, mf.flushCount)

		_, err = fw.Write([]byte(" second"))
		require.NoError(t, err)
		assert.Equal(t, 2, mf.flushCount)

		assert.Equal(t, "first second", mf.String())
	})

	t.Run("returns write error", func(t *testing.T) {
		// Create a writer that fails
		failWriter := &errorWriter{err: errors.New("write failed")}
		fw := NewFlushingWriter(failWriter)

		_, err := fw.Write([]byte("test"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "write failed")
	})

	t.Run("returns flush error", func(t *testing.T) {
		mf := &mockFlusher{
			flushError: errors.New("flush failed"),
		}
		fw := NewFlushingWriter(mf)

		_, err := fw.Write([]byte("test"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "flush failed")
	})
}

func TestFlushingWriter_Flush(t *testing.T) {
	t.Run("explicit flush", func(t *testing.T) {
		mf := &mockFlusher{}
		fw := NewFlushingWriter(mf)

		err := fw.Flush()
		require.NoError(t, err)
		assert.Equal(t, 1, mf.flushCount)
	})

	t.Run("flush with error", func(t *testing.T) {
		mf := &mockFlusher{
			flushError: errors.New("flush error"),
		}
		fw := NewFlushingWriter(mf)

		err := fw.Flush()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "flush error")
	})
}

// errorWriter is a writer that always returns an error
type errorWriter struct {
	err error
}

func (e *errorWriter) Write(_ []byte) (n int, err error) {
	return 0, e.err
}
