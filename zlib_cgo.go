package zlib

/*

#cgo CFLAGS: -DHAS_PCLMUL -mpclmul -DHAS_SSE42 -msse4.2 -D_LARGEFILE64_SOURCE=1 -DHAVE_HIDDEN

#include <errno.h>
#include "./zlib.h"

extern int zs_inflate_init(char* stream);
extern int zs_inflate(char* stream, void* in, int* in_bytes, void* out, int* out_bytes);
extern int zs_inflate_avail_in(char* stream);
extern int zs_inflate_avail_out(char* stream);
extern int zs_get_errno();
void zs_inflate_end(char *stream);

*/
import "C"

import (
	"fmt"
	"io"
	"unsafe"

	"errors"

	"golang.org/x/sys/unix"
)

type zstream [unsafe.Sizeof(C.z_stream{})]C.char

type reader struct {
	in      io.Reader
	zs      zstream
	inBuf   []byte
	inAvail []byte // part of inBuf that's yet to be inflated
	err     error
	rErr    error // Error from r. Maybe io.EOF
}

const defaultBufferSize = 1 << 20

// NewReaderBuffer creates a gzip reader with default settings.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	return NewReaderBuffer(r, defaultBufferSize)
}

// NewReaderBuffer creates a new gzip reader with a given prefetch buffer size.
func NewReaderBuffer(r io.Reader, bufSize int) (io.ReadCloser, error) {
	z := &reader{
		inBuf: make([]byte, bufSize),
	}
	ec := C.zs_inflate_init(&z.zs[0])
	if ec != 0 {
		panic(ec)
	}
	return z, nil
}

// Close implements io.Closer.
func (z *reader) Close() error {
	C.zs_inflate_end(&z.zs[0])
	return z.err
}

// Read implements io.Reader.
func (z *reader) Read(out []byte) (int, error) {
	var orgOut = out
	for z.err == nil && len(out) > 0 {
		if len(z.inAvail) == 0 && z.rErr == nil {
			var n int
			n, z.rErr = z.in.Read(z.inBuf)
			if n > 0 {
				z.inAvail = z.inBuf[:n]
			}
			if z.rErr != nil && z.rErr != io.EOF {
				z.err = z.rErr
				break
			}
		}
		var (
			inPtr unsafe.Pointer
			inLen C.int
		)
		if inLen = C.int(len(z.inAvail)); inLen > 0 {
			inPtr = unsafe.Pointer(&z.inAvail[0])
		}
		outLen := C.int(len(out))
		ret := C.zs_inflate(&z.zs[0], inPtr, &inLen, unsafe.Pointer(&out[0]), &outLen)
		if ret == C.Z_STREAM_END {
			ret = C.Z_STREAM_ERROR
		}
		if ret != 0 {
			z.err = zlibReturnCodeToError(ret)
			break
		}
		nIn := len(z.inAvail) - int(inLen)
		nOut := len(out) - int(outLen)
		if nIn == 0 && nOut == 0 { // got to make some progress
			panic("stuck")
		}
		out = out[nOut:]
		z.inAvail = z.inAvail[nIn:]
		if len(out) > 0 && z.rErr != nil {
			// Consumed all the input data
			if inAvail := C.zs_inflate_avail_in(&z.zs[0]); inAvail != 0 {
				panic(inAvail)
			}
			z.err = z.rErr
			break
		}
	}
	return len(orgOut) - len(out), z.err
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
