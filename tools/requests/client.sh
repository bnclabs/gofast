#! /usr/bin/env bash

# client.sh <server:addr>

echo "building client ..."
go build;

echo "time taken...."
time ./requests -addr $1 -conns 16 -routines 10 -count 10000
echo

echo "building client profile information ..."
go tool pprof -svg requests requests.pprof > server.pprof.svg
go tool pprof -inuse_space -svg requests requests.mprof > server.ispace.svg
go tool pprof -inuse_objects -svg requests requests.mprof > server.iobjs.svg
go tool pprof -alloc_space -svg requests requests.mprof > server.aspace.svg
go tool pprof -alloc_objects -svg requests requests.mprof > server.aobjs.svg
echo

