#! /usr/bin/env bash

# client.sh <server:addr>

echo "building ..."
go build;

# -do can be post, request
time ./perf -profile -c -addr $1 -do $2 -payload 64 -batchsize 200 -conns 16 -routines 200 -count 100000

# streamrx, streamtx
# time ./perf -c -addr $1 -do $2 -batchsize 1000 -payload 16000 -buffersize 17000 -conns 1 -routines 1 -count 1 -stream 100000

# verify
# time ./perf -c -addr $1 -do verify -batchsize 200 -conns 16 -routines 200 -count 200000 -stream 100

echo

echo "building client profile information ..."
go tool pprof -svg perf client.pprof > client.pprof.svg
go tool pprof -inuse_space -svg perf client.mprof > client.ispace.svg
go tool pprof -inuse_objects -svg perf client.mprof > client.iobjs.svg
go tool pprof -alloc_space -svg perf client.mprof > client.aspace.svg
go tool pprof -alloc_objects -svg perf client.mprof > client.aobjs.svg
echo
