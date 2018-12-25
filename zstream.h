#ifndef ZSTREAM_H
#define ZSTREAM_H

extern int zs_inflate_init(char* stream);
extern int zs_inflate(char* stream, void* out, int* out_bytes,
                      int* consumed_input);
extern int zs_inflate_with_input(char* stream, void* in, int in_bytes,
                                 void* out, int* out_bytes,
                                 int* consumed_input);

extern int zs_deflate_with_input(char* stream, void* in, int in_bytes, void* out,
                          int* out_bytes);
extern int zs_deflate_init(char* stream, int level);
extern int zs_deflate(char* stream, void* out, int* out_bytes);
extern int zs_deflate_end(char* stream, void* out, int* out_bytes);
extern int zs_inflate_avail_in(char* stream);
extern int zs_inflate_avail_out(char* stream);

extern int zs_get_errno();
void zs_inflate_end(char* stream);

#endif /* ZSTREAM_H */
