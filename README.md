[![GoDoc](https://godoc.org/github.com/yasushi-saito/cloudflare-zlib?status.svg)](https://godoc.org/github.com/yasushi-saito/cloudflare-zlib)


Go bindings for cloudflare zlib fork (https://github.com/cloudflare/zlib).

## Status

As of 2018-12-24

- Only tested for Linux amd64 + cgo.
- You need to set

```
export CGO_CFLAGS_ALLOW=-m.*
```

  before building.


## Benchmark

As of 2018-12-24.

File: 240MB text file.
The file is compressed 5.2x using any of the systems tested.

```
BenchmarkInflateStandardGzip-8     	     100	  18325651 ns/op
BenchmarkInflateKlauspostGzip-8    	     100	  21908492 ns/op
BenchmarkInflateCloudflareGzip-8   	     200	   7312906 ns/op
BenchmarkDeflateStandardGzip-8     	       1	11806820047 ns/op
BenchmarkDeflateKlauspostGzip-8    	       1	3039031194 ns/op
BenchmarkDeflateCloudflareGzip-8   	       1	3033015128 ns/op
```
