[![GoDoc](https://godoc.org/github.com/yasushi-saito/cloudflare-zlib?status.svg)](https://godoc.org/github.com/yasushi-saito/cloudflare-zlib)


Go bindings for cloudflare zlib fork (https://github.com/cloudflare/zlib).

## Status

As of 2018-12-24

- only tested for Linux amd64 + cgo.
- only the reader is implemented.


## Benchmark

As of 2018-12-24

It's over 2x faster than the standard implementation.

```
BenchmarkInflateStandardGzip-8     	     100	  16950147 ns/op
BenchmarkInflateKlauspostGzip-8    	     100	  20263415 ns/op
BenchmarkInflateCloudflareGzip-8   	     200	   6976053 ns/op
```
