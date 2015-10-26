#! /usr/bin/env bash

# client.sh <server:addr>

echo "building client ..."
go build;

echo "time taken...."
time ./client -addr $1 -conns 16 -routines 200 -count 10000
echo

echo "building client profile information ..."
go tool pprof -svg client client.pprof > client.pprof.svg
go tool pprof -inuse_space -svg client client.mprof > client.ispace.svg
go tool pprof -inuse_objects -svg client client.mprof > client.iobjs.svg
go tool pprof -alloc_space -svg client client.mprof > client.aspace.svg
go tool pprof -alloc_objects -svg client client.mprof > client.aobjs.svg
echo

