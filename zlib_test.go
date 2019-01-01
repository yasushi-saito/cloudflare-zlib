package zlib_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/grailbio/testutil/assert"
	kgzip "github.com/klauspost/compress/gzip"
	zlib "github.com/yasushi-saito/cloudflare-zlib"
)

func testInflate(t *testing.T, r *rand.Rand, src []byte, want []byte) {
	zin, err := zlib.NewReader(bytes.NewReader(src))
	assert.NoError(t, err)

	var (
		got []byte
		buf = make([]byte, 8192)
	)

	noProgress := 0
	iter := 0
	for {
		iter++
		n := rand.Intn(8192)
		n2, err := zin.Read(buf[:n])
		if n2 > 0 {
			got = append(got, buf[:n2]...)
			noProgress = 0
		} else {
			noProgress++
			assert.LT(t, noProgress, 2, "iter=%d", iter)
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
		n := r.Intn(16 << 20) + 1
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

// Test packed gzip
func TestInflateRandomPacked(t *testing.T) {
	r := rand.New(rand.NewSource(0))
	for i := 0; i < 20; i++ {
		compressed := bytes.Buffer{}
		uncompressed := bytes.Buffer{}

		log.Printf("%d", i)
		for j := 0; j < 10; j++ {
			n := r.Intn(2 << 20) + 1
			buf := make([]byte, n)
			_, err := r.Read(buf)
			assert.NoError(t, err)
			uncompressed.Write(buf)

			gz := gzip.NewWriter(&compressed)
			_, err = gz.Write(buf)
			assert.NoError(t, err)
			assert.NoError(t, gz.Close())
		}
		testInflate(t, r, compressed.Bytes(), uncompressed.Bytes())
	}
}

func testDeflate(t *testing.T, r *rand.Rand, src []byte) {
	orgSrc := src
	out := bytes.Buffer{}
	zout, err := zlib.NewWriter(&out)
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

var (
	testPathFlag = flag.String("path",
		"/home/ysaito/CNVS-NORM-110033752-cfDNA-WGBS-Rep1_S1_L001_R1_001.fastq", "Plain-text file used for in tests and benchmarks")
	testGZPathFlag = flag.String("gz-path",
		"/scratch-nvme/cache_tmp/170206_ARTLoD_B1_01rerun_S1_L001_R1_001.fastq.gz",
		"Gzipped file used in tests and benchmarks")
	runManualTestsFlag = flag.Bool("run-manual-tests",
		false, "Run large tests using files outside the repo")
)

func TestDeflateLarge(t *testing.T) {
	if !*runManualTestsFlag {
		t.Skip("--run-manual-tests not set")
	}
	testDeflateLarge(t, *testGZPathFlag)
}

func testDeflateLarge(t *testing.T, gzPath string) {
	type reader struct {
		in             *os.File
		r              io.Reader
		buf, remaining []byte
	}
	const bufSize = 1 << 20
	var (
		err    error
		r0, r1 reader
		r      = rand.New(rand.NewSource(0))
	)
	open := func(r *reader) {
		r.in, err = os.Open(*testGZPathFlag)
		assert.NoError(t, err)
		r.buf = make([]byte, bufSize)
	}
	read := func(r *reader, want int) ([]byte, bool) {
		buf := make([]byte, want)
		remaining := buf
		for {
			n := len(remaining)
			if n > len(r.remaining) {
				n = len(r.remaining)
			}
			copy(remaining, r.remaining)
			remaining = r.remaining[n:]
			r.remaining = r.remaining[n:]
			if len(remaining) == 0 {
				break
			}
			got, err := r.r.Read(r.buf)
			if got == 0 {
				assert.EQ(t, err, io.EOF)
				break
			}
			if err != nil {
				assert.EQ(t, err, io.EOF)
			}
			r.buf = r.buf[got:]
			r.remaining = r.buf
		}
		if len(remaining) == want {
			return nil, false
		}
		return buf[0 : len(buf)-len(remaining)], true
	}

	open(&r0)
	r0.r, err = gzip.NewReader(r0.in)
	assert.NoError(t, err)
	open(&r1)
	r1.r, err = zlib.NewReader(r1.in)
	assert.NoError(t, err)

	total := 0
	last := 0
	for {
		nMax := r.Intn(bufSize)
		buf0, ok0 := read(&r0, nMax)
		buf1, ok1 := read(&r1, nMax)
		if !bytes.Equal(buf0, buf1) {
			t.Fatalf("want %d gotn0 %d gotn1 %d", nMax, len(buf0), len(buf1))
		}
		assert.EQ(t, ok0, ok1)
		if !ok0 {
			break
		}
		total += len(buf0)
		if total-last > 1<<30 {
			log.Printf("read %d bytes", total)
			last = total
		}
	}
	assert.NoError(t, r0.in.Close())
	assert.NoError(t, r1.in.Close())
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

func BenchmarkInflateZlib(b *testing.B) {
	benchmarkInflate(b, "/tmp/get-pip.py",
		func(in io.Reader) (io.Reader, error) {
			return zlib.NewReaderBuffer(in, 512<<10)
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
}

func BenchmarkDeflateStandardGzip(b *testing.B) {
	benchmarkDeflate(b, *testPathFlag,
		func(out io.Writer) io.WriteCloser {
			w, err := gzip.NewWriterLevel(out, 5)
			assert.NoError(b, err)
			return w
		})
}

func BenchmarkDeflateKlauspostGzip(b *testing.B) {
	benchmarkDeflate(b, *testPathFlag,
		func(out io.Writer) io.WriteCloser {
			w, err := kgzip.NewWriterLevel(out, 5)
			assert.NoError(b, err)
			return w
		})
}

func BenchmarkDeflateZlib(b *testing.B) {
	benchmarkDeflate(b, *testPathFlag,
		func(out io.Writer) io.WriteCloser {
			w, err := zlib.NewWriterLevel(out, 5, 512<<10)
			assert.NoError(b, err)
			return w
		})
}
