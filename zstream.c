#include "./zstream.h"
#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "./zlib.h"

int zs_inflate_init(char* stream) {
  z_stream* zs = (z_stream*)stream;
  memset(zs, 0, sizeof(*zs));
  // 16 makes it understand only gzip files
  return z_inflateInit2_(zs, 16 + 15, ZLIB_VERSION, sizeof(*zs));
}

void zs_inflate_end(char* stream) { z_inflateEnd((z_stream*)stream); }

int zs_inflate_reset(char* stream) {
  z_stream* zs = (z_stream*)stream;
  return z_inflateReset(zs);
}

int zs_get_errno() { return errno; }

int zs_inflate(char* stream, void* in, int in_bytes, void* out, int* out_bytes,
               int* consumed_input) {
  z_stream* zs = (z_stream*)stream;
  if (in_bytes > 0) {
    if (zs->avail_in != 0) {
      abort();
    }
    zs->avail_in = in_bytes;
    zs->next_in = in;
  } else {
    if (zs->avail_in == 0) {
      abort();
    }
  }
  zs->next_out = out;
  zs->avail_out = *out_bytes;
  int ret = z_inflate((z_stream*)stream, Z_NO_FLUSH);
  if (ret == Z_OK || ret == Z_STREAM_END) {
    *out_bytes = zs->avail_out;
  }
  *consumed_input = (zs->avail_in == 0);
  return ret;
}

int zs_deflate_init(char* stream, int level) {
  z_stream* zs = (z_stream*)stream;
  memset(zs, 0, sizeof(*zs));
  return deflateInit2(zs, level, Z_DEFLATED, 16 + 15, 8, Z_DEFAULT_STRATEGY);
}

int zs_deflate(char* stream, void* in, int in_bytes, void* out,
               int* out_bytes) {
  z_stream* zs = (z_stream*)stream;
  if (in_bytes > 0) {
    if (zs->avail_in != 0) {  // has buffered input
      abort();
    }
    zs->avail_in = in_bytes;
    zs->next_in = in;
  } else if (zs->avail_in == 0) {
    abort();
  }
  zs->next_out = out;
  zs->avail_out = *out_bytes;
  int ret = z_deflate(zs, Z_NO_FLUSH);
  *out_bytes = zs->avail_out;
  return ret;
}

int zs_deflate_end(char* stream, void* out, int* out_bytes) {
  z_stream* zs = (z_stream*)stream;
  if (zs->avail_in != 0) {
    abort();
  }
  zs->next_out = out;
  zs->avail_out = *out_bytes;
  int ret = z_deflate(zs, Z_FINISH);
  *out_bytes = zs->avail_out;
  if (ret != Z_OK) {
    z_deflateEnd(zs);
  }
  return ret;
}
