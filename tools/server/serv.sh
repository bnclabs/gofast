#! /usr/bin/env bash

# serv.sh <server:addr>

echo "building server ..."
go build

echo "starting server $1 ..."
./server -addr $1

echo "building server profile information ..."
cd ../server
go tool pprof -svg server server.pprof > server.pprof.svg
go tool pprof -inuse_space -svg server server.mprof > server.ispace.svg
go tool pprof -inuse_objects -svg server server.mprof > server.iobjs.svg
go tool pprof -alloc_space -svg server server.mprof > server.aspace.svg
go tool pprof -alloc_objects -svg server server.mprof > server.aobjs.svg
