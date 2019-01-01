// +build cgo,amd64

package zlib

/*

#cgo CFLAGS: -march=ivybridge -std=c99 -Wall -D_LARGEFILE64_SOURCE=1 -DHAVE_HIDDEN -DHAVE_INTERNAL -DHAVE_BUILTIN_CTZL -DMEDIUM_STRATEGY -DX86_64 -DX86_NOCHECK_SSE2 -DUNALIGNED_OK -DUNROLL_LESS -DX86_CPUID -DX86_SSE2_FILL_WINDOW -DX86_SSE4_2_CRC_HASH -DX86_SSE4_2_CRC_INTRIN -DX86_PCLMULQDQ_CRC -DX86_QUICK_STRATEGY -I.

#include <errno.h>
#include "./zlib.h"
#include "./zstream.h"

*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"unsafe"

	"golang.org/x/sys/unix"
)

type zstream [unsafe.Sizeof(C.z_stream{})]C.char

type reader struct {
	in         io.Reader
	inConsumed bool    // true if zstream has finished consuming the current input buffer.
	inEOF      bool    // true if in reaches io.EOF
	zs         zstream // underlying zlib implementation.
	inBuf      []byte
	err        error
}

// defaultBufferSize is the default buffer size used by NewBuffer.
const defaultBufferSize = 512 * 1024

// NewReader creates a gzip reader with 512KB buffer.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	return NewReaderBuffer(r, defaultBufferSize)
}

// NewReaderBuffer creates a new gzip reader with a given prefetch buffer size.
func NewReaderBuffer(in io.Reader, bufSize int) (io.ReadCloser, error) {
	z := &reader{
		in:         in,
		inBuf:      make([]byte, bufSize),
		inConsumed: true, // force in.Read
	}
	ec := C.zs_inflate_init(&z.zs[0])
	if ec != 0 {
		return nil, zlibReturnCodeToError(ec)
	}
	return z, nil
}

// Close implements io.Closer.
func (z *reader) Close() error {
	C.zs_inflate_end(&z.zs[0])
	if z.err == io.EOF {
		return nil
	}
	return z.err
}

// Read implements io.Reader.
func (z *reader) Read(out []byte) (int, error) {
	var orgOut = out
	for z.err == nil && len(out) > 0 {
		var (
			outLen     = C.int(len(out))
			ret        C.int
			inConsumed C.int
		)
		if !z.inConsumed {
			ret = C.zs_inflate(&z.zs[0], nil, 0, unsafe.Pointer(&out[0]), &outLen, &inConsumed)
		} else {
			if z.inEOF {
				z.err = io.EOF
				break
			}
			n, err := z.in.Read(z.inBuf)
			if err != nil {
				if err != io.EOF {
					z.err = err
					break
				}
				z.inEOF = true
				// fall through
			}
			if n == 0 {
				if !z.inEOF {
					panic(z)
				}
				z.err = io.EOF
				break
			}
			ret = C.zs_inflate(&z.zs[0], unsafe.Pointer(&z.inBuf[0]), C.int(n), unsafe.Pointer(&out[0]), &outLen, &inConsumed)
		}
		z.inConsumed = (inConsumed != 0)
		if ret != C.Z_STREAM_END && ret != C.Z_OK {
			z.err = zlibReturnCodeToError(ret)
			break
		}
		nOut := len(out) - int(outLen)
		out = out[nOut:]
		if ret == C.Z_STREAM_END {
			ret = C.zs_inflate_reset(&z.zs[0])
			if ret != C.Z_OK {
				z.err = zlibReturnCodeToError(ret)
			}
			break
		}
	}
	return len(orgOut) - len(out), z.err
}

type writer struct {
	out    io.Writer
	zs     zstream // underlying zlib implementation.
	outBuf []byte
	err    error
}

// NewWriter creates a gzip writer with default settings.
func NewWriter(w io.Writer) (io.WriteCloser, error) {
	return NewWriterLevel(w, -1, defaultBufferSize)
}

// NewWriterLevel creates a gzip writer. Level is the compression level; -1
// means the default level. bufSize is the internal buffer size. It defaults to
// 512KB.
func NewWriterLevel(w io.Writer, level int, bufSize int) (io.WriteCloser, error) {
	z := &writer{
		out:    w,
		outBuf: make([]byte, bufSize),
	}
	ec := C.zs_deflate_init(&z.zs[0], C.int(level))
	if ec != 0 {
		return nil, zlibReturnCodeToError(ec)
	}
	return z, nil
}

func (z *writer) push(data []byte) error {
	n, err := z.out.Write(data)
	if err != nil {
		return err
	}
	if n < len(data) { // shouldn't happen in practice
		return fmt.Errorf("zlib: n=%d, outLen=%d", n, len(data))
	}
	return nil
}

// Close implements io.Closer
func (z *writer) Close() error {
	for {
		outLen := C.int(len(z.outBuf))
		ret := C.zs_deflate_end(&z.zs[0], unsafe.Pointer(&z.outBuf[0]), &outLen)
		if ret != 0 && ret != C.Z_STREAM_END {
			return zlibReturnCodeToError(ret)
		}
		nOut := len(z.outBuf) - int(outLen)
		if err := z.push(z.outBuf[:nOut]); err != nil {
			return err
		}
		if ret == C.Z_STREAM_END {
			return nil
		}
	}
}

// Write implements io.Writer.
func (z *writer) Write(in []byte) (int, error) {
	if len(in) == 0 {
		return 0, nil
	}
	var outLen = C.int(len(z.outBuf))
	ret := C.zs_deflate(&z.zs[0], unsafe.Pointer(&in[0]), C.int(len(in)),
		unsafe.Pointer(&z.outBuf[0]), &outLen)
	if ret != 0 {
		return 0, zlibReturnCodeToError(ret)
	}
	nOut := len(z.outBuf) - int(outLen)
	if err := z.push(z.outBuf[:nOut]); err != nil {
		return 0, err
	}
	if outLen > 0 { // outbuf didn't fillup, i.e., the input was fully consumed.
		return len(in), nil
	}
	for {
		outLen = C.int(len(z.outBuf))
		ret = C.zs_deflate(&z.zs[0], nil, 0, unsafe.Pointer(&z.outBuf[0]), &outLen)
		if ret != 0 {
			return 0, zlibReturnCodeToError(ret)
		}
		nOut := len(z.outBuf) - int(outLen)
		if err := z.push(z.outBuf[:nOut]); err != nil {
			return 0, err
		}
		if outLen > 0 { // outbuf didn't fillup, i.e., the input was fully consumed.
			break
		}
	}
	return len(in), nil
}

var zlibErrors = map[C.int]error{
	C.Z_OK:            nil,
	C.Z_STREAM_END:    io.EOF,
	C.Z_ERRNO:         nil, // handled separately
	C.Z_STREAM_ERROR:  errors.New("Zlib: stream error"),
	C.Z_DATA_ERROR:    errors.New("Zlib: data error"),
	C.Z_MEM_ERROR:     errors.New("Zlib: mem error"),
	C.Z_BUF_ERROR:     errors.New("Zlib: buf error"),
	C.Z_VERSION_ERROR: errors.New("Zlib: version error"),
}

func zlibReturnCodeToError(r C.int) error {
	if r == 0 {
		return nil
	}
	if r == C.Z_ERRNO {
		return unix.Errno(C.zs_get_errno())
	}
	if err, ok := zlibErrors[r]; ok {
		return err
	}
	return fmt.Errorf("Zlib: unknown error %d", r)
}
