#! /usr/bin/env bash

# client.sh <server:addr>

rm -rf example
echo "building ..."
go build;

# -do can be post, request
time ./example -do post -batchsize 200 -conns 16 -routines 100 -count 100000 -payload 50000 -buffersize 51000

go tool pprof -inuse_space -svg example example.mprof > example.ispace.svg
go tool pprof -alloc_space -svg example example.mprof > example.aspace.svg
