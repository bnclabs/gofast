#! /usr/bin/env bash

# serv.sh <server:addr>

echo "building ..."
go build;

echo "starting server $1 ..."
./perf -profile -s -addr $1 -batchsize 16 -routines 200

# streamrx, streamtx
# ./perf -s -addr $1 -batchsize 1000 -payload 16000 -buffersize 17000 -stream 100000 -routines 200

# verify
# ./perf -s -addr $1 -do verify -batchsize 200 -stream 100 -routines 200


echo

echo "building server profile information ..."
go tool pprof -svg perf server.pprof > server.pprof.svg
go tool pprof -inuse_space -svg perf server.mprof > server.ispace.svg
go tool pprof -inuse_objects -svg perf server.mprof > server.iobjs.svg
go tool pprof -alloc_space -svg perf server.mprof > server.aspace.svg
go tool pprof -alloc_objects -svg perf server.mprof > server.aobjs.svg
