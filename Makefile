build:
	go build

test:
	go test -v -race -timeout 4000s -test.run=. -test.bench=. -test.benchmem=true

bench:
	go test -v -timeout 4000s -test.run=. -test.bench=. -test.benchmem=true

heaptest:
	go test -gcflags '-m -l' -timeout 4000s -test.run=. -test.bench=. -test.benchmem=true > escapel 2>&1
	grep "^\.\/.*escapes to heap" escapel | tee escapelines
	grep panic *.go | tee -a escapelines

coverage:
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm -rf coverage.out

clean:
	rm -f tools/client/{client,client.mprof,client.pprof,*.svg}
	rm -f tools/server/{server,server.mprof,server.pprof,*.svg}
	rm -f tools/{*.mprof,*.pprof,*.svg}
	rm -rf escapel escapem escapelines example/example example/*.svg
