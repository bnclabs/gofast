#! /usr/bin/env bash

# client.sh <server:addr>

echo "building client ..."
go build;

echo "time taken...."
time ./requests -addr $1 -conns 16 -routines 100 -count 10000
echo

echo "building client profile information ..."
go tool pprof -svg requests requests.pprof > requests.pprof.svg
go tool pprof -inuse_space -svg requests requests.mprof > requests.ispace.svg
go tool pprof -inuse_objects -svg requests requests.mprof > requests.iobjs.svg
go tool pprof -alloc_space -svg requests requests.mprof > requests.aspace.svg
go tool pprof -alloc_objects -svg requests requests.mprof > requests.aobjs.svg
echo

