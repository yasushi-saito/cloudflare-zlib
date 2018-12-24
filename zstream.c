#include "./zstream.h"
#include <errno.h>
#include <stdlib.h>
#include "./zlib.h"

int zs_inflate_init(char* stream) {
  z_streamp zs = (z_streamp)stream;
  zs->zalloc = Z_NULL;
  zs->zfree = Z_NULL;
  zs->opaque = Z_NULL;

  // 16 makes it understand only gzip files
  return z_inflateInit2_(zs, 16 + 15, ZLIB_VERSION, sizeof(*zs));
}

void zs_inflate_end(char* stream) { z_inflateEnd((z_streamp)stream); }

int zs_inflate_avail_in(char* stream) {
  z_streamp zs = (z_streamp)stream;
  return zs->avail_in;
}

int zs_inflate_avail_out(char* stream) {
  z_streamp zs = (z_streamp)stream;
  return zs->avail_out;
}

int zs_get_errno() { return errno; }

int zs_inflate_with_input(char* stream, void* in, int in_bytes, void* out,
                          int* out_bytes, int* consumed_input) {
  z_streamp zs = (z_streamp)stream;
  if (zs->avail_in != 0 || in_bytes <= 0) {
    abort();
  }
  zs->avail_in = in_bytes;
  zs->next_in = in;
  zs->next_out = out;
  zs->avail_out = *out_bytes;
  int ret = z_inflate((z_streamp)stream, Z_NO_FLUSH);
  if (ret == Z_OK || ret == Z_STREAM_END) {
    *out_bytes = zs->avail_out;
  }
  *consumed_input = (zs->avail_in == 0);
  return ret;
}

int zs_inflate(char* stream, void* out, int* out_bytes, int* consumed_input) {
  z_streamp zs = (z_streamp)stream;
  if (zs->avail_in == 0) {
    abort();
  }
  zs->next_out = out;
  zs->avail_out = *out_bytes;
  int ret = z_inflate((z_streamp)stream, Z_NO_FLUSH);
  if (ret == Z_OK || ret == Z_STREAM_END) {
    *out_bytes = zs->avail_out;
  }
  *consumed_input = (zs->avail_in == 0);
  return ret;
}
