package zlib_test

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"math/rand"

	"compress/gzip"

	"bytes"

	"io/ioutil"

	"github.com/grailbio/testutil/assert"
	czlib "github.com/yasushi-saito/cloudflare-zlib"
	kgzip "github.com/klauspost/compress/gzip"
)

func testInflate(t *testing.T, r *rand.Rand, src []byte, want []byte) {
	zin, err := czlib.NewReader(bytes.NewReader(src))
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

func benchmarkInflate(
	b *testing.B,
	path string,
	inflateFactory func(in io.Reader) (io.Reader, error)) {
	b.StopTimer()
	tmp, err := ioutil.TempDir("", "")
	assert.NoError(b, err)
	defer os.RemoveAll(tmp)

	in, err := os.Open(path)
	assert.NoError(b, err)
	dstPath := filepath.Join(tmp, "tmp.gz")
	out, err := os.Create(dstPath)
	assert.NoError(b, err)
	outgz := gzip.NewWriter(out)
	wantByte, err := io.Copy(outgz, in)
	assert.NoError(b, err)
	assert.NoError(b, outgz.Close())
	assert.NoError(b, in.Close())
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		in, err = os.Open(dstPath)
		assert.NoError(b, err)
		inflator, err := inflateFactory(in)
		assert.NoError(b, err)
		n, err := io.Copy(ioutil.Discard, inflator)
		assert.NoError(b, err)
		assert.EQ(b, n, wantByte)
	}
}

func BenchmarkInflateStandardGzip(b *testing.B) {
	benchmarkInflate(b, "/tmp/get-pip.py",
		func(in io.Reader) (io.Reader, error) {
			return gzip.NewReader(in)
		})
}

func BenchmarkInflateKlauspostGzip(b *testing.B) {
	benchmarkInflate(b, "/tmp/get-pip.py",
		func(in io.Reader) (io.Reader,error) {
			return kgzip.NewReader(in)
		})
}

func BenchmarkInflateCloudflareGzip(b *testing.B) {
	benchmarkInflate(b, "/tmp/get-pip.py",
		func(in io.Reader) (io.Reader,error) {
			return czlib.NewReader(in)
		})
}
