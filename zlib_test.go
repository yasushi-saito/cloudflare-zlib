package zlib_test

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"math/rand"

	"github.com/grailbio/testutil/assert"
	"github.com/yasushi-saito/cloudflare-zlib"
)

func testInflate(t *testing.T, r *rand.Rand, srcPath string, want []byte) {
	in, err := os.Open(srcPath)
	assert.NoError(t, err)
	zin, err := zlib.NewReader(in)
	assert.NoError(t, err)

	var got []byte
	for {
		buf := make([]byte, rand.Intn(8192))
		n, err := zin.Read(buf)
		if n > 0 {
			got = append(got, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
	}
	assert.NoError(t, zin.Close())
	assert.NoError(t, in.Close())
	assert.EQ(t, got, want)
}

func TestInflate(t *testing.T) {
	r := rand.New(rand.NewSource(0))

	want, err := ioutil.ReadFile("deflate.c")
	assert.NoError(t, err)
	testInflate(t, r, "./test.txt.gz", want)
}
