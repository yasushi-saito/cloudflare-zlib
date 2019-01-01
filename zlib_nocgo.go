// +build !cgo !amd64

package zlib

import (
	"io"

	"github.com/klauspost/compress/gzip"
)

// NewReader creates a gzip reader with 512KB buffer.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

// NewReaderBuffer creates a new gzip reader with a given prefetch buffer size.
func NewReaderBuffer(r io.Reader, bufSize int) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

// NewWriter creates a gzip writer with default settings.
func NewWriter(w io.Writer) (io.WriteCloser, error) {
	return gzip.NewWriterLevel(w, -1)
}

// NewWriterLevel creates a gzip writer. Level is the compression level; -1
// means the default level. bufSize is the internal buffer size. It defaults to
// 512KB.
func NewWriterLevel(w io.Writer, level int, bufSize int) (io.WriteCloser, error) {
	return gzip.NewWriterLevel(w, level)
}
