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

	"bufio"

	"github.com/grailbio/testutil/assert"
	kgzip "github.com/klauspost/compress/gzip"
	czlib "github.com/yasushi-saito/cloudflare-zlib"
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

func testDeflate(t *testing.T, r *rand.Rand, src []byte) {
	orgSrc := src
	out := bytes.Buffer{}
	zout, err := czlib.NewWriter(&out)
	assert.NoError(t, err)

	for len(src) > 0 {
		n := r.Intn(8192)
		if n > len(src) {
			n = len(src)
		}
		n2, err := zout.Write(src[:n])
		assert.NoError(t, err)
		assert.EQ(t, n, n2)
		src = src[n:]
	}
	assert.NoError(t, zout.Close())

	got := bytes.Buffer{}
	zin, err := gzip.NewReader(bytes.NewReader(out.Bytes()))
	assert.NoError(t, err)
	n, err := io.Copy(&got, zin)
	assert.NoError(t, err)
	assert.EQ(t, int(n), len(orgSrc))
	if !bytes.Equal(got.Bytes(), orgSrc) {
		t.Fatal("fail")
	}
}

func TestDeflateRandom(t *testing.T) {
	r := rand.New(rand.NewSource(0))
	for i := 0; i < 20; i++ {
		n := r.Intn(16 << 20)
		log.Printf("%d: n=%d", i, n)
		data := make([]byte, n)
		_, err := r.Read(data)
		assert.NoError(t, err)
		testDeflate(t, r, data)
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
		func(in io.Reader) (io.Reader, error) {
			return kgzip.NewReader(in)
		})
}

func BenchmarkInflateCloudflareGzip(b *testing.B) {
	benchmarkInflate(b, "/tmp/get-pip.py",
		func(in io.Reader) (io.Reader, error) {
			return czlib.NewReaderBuffer(in, 512<<10)
		})
}

type discardingWriter struct {
	n int64
}

func (w *discardingWriter) Write(data []byte) (int, error) {
	w.n += int64(len(data))
	return len(data), nil
}

func benchmarkDeflate(
	b *testing.B,
	path string,
	deflateFactory func(out io.Writer) io.WriteCloser) {
	var w discardingWriter
	for i := 0; i < b.N; i++ {
		w = discardingWriter{}
		deflator := deflateFactory(&w)
		in, err := os.Open(path)
		assert.NoError(b, err)
		_, err = io.Copy(deflator, bufio.NewReaderSize(in, 1<<20))
		assert.NoError(b, err)
		assert.NoError(b, deflator.Close())
		assert.NoError(b, in.Close())
	}
	b.Logf("Compressed size: %d", w.n)
}

// var benchPath = "/tmp/get-pip.py"
var benchPath = "/home/ysaito/CNVS-NORM-110033752-cfDNA-WGBS-Rep1_S1_L001_R1_001.fastq"

func BenchmarkDeflateStandardGzip(b *testing.B) {
	benchmarkDeflate(b, benchPath,
		func(out io.Writer) io.WriteCloser {
			w, err := gzip.NewWriterLevel(out, 5)
			assert.NoError(b, err)
			return w
		})
}

func BenchmarkDeflateKlauspostGzip(b *testing.B) {
	benchmarkDeflate(b, benchPath,
		func(out io.Writer) io.WriteCloser {
			w, err := kgzip.NewWriterLevel(out, 5)
			assert.NoError(b, err)
			return w
		})
}

func BenchmarkDeflateCloudflareGzip(b *testing.B) {
	benchmarkDeflate(b, benchPath,
		func(out io.Writer) io.WriteCloser {
			w, err := czlib.NewWriterLevel(out, 5, 512<<10)
			assert.NoError(b, err)
			return w
		})
}
