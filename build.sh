#!/bin/sh
export CGO_CFLAGS_ALLOW=-m.*
go test .
