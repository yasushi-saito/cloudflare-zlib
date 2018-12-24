package zlib_test

import (
	"io"
	"log"
	"testing"

	"math/rand"

	"compress/gzip"

	"bytes"

	"github.com/grailbio/testutil/assert"
	"github.com/yasushi-saito/cloudflare-zlib"
)

func testInflate(t *testing.T, r *rand.Rand, src []byte, want []byte) {
	zin, err := zlib.NewReader(bytes.NewReader(src))
	assert.NoError(t, err)

	var (
		got []byte
		buf = make([]byte, 8192)
	)

	for {
		n := rand.Intn(8192)
		n2, err := zin.Read(buf[:n])
		if n2 > 0 {
			got = append(got, buf[:n2]...)
		} else if n > 0 {
			assert.NotNil(t, err)
		}
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
	}
	assert.NoError(t, zin.Close())
	if !bytes.Equal(got, want) {
		t.Fatal("fail")
	}
}

func TestInflateRandom(t *testing.T) {
	r := rand.New(rand.NewSource(0))
	for i := 0; i < 20; i++ {
		n := r.Intn(16 << 20)
		log.Printf("%d: n=%d", i, n)
		uncompressed := make([]byte, n)
		_, err := r.Read(uncompressed)
		assert.NoError(t, err)

		compressed := bytes.Buffer{}
		gz := gzip.NewWriter(&compressed)
		_, err = gz.Write(uncompressed)
		assert.NoError(t, err)
		assert.NoError(t, gz.Close())
		testInflate(t, r, compressed.Bytes(), uncompressed)
	}
}
