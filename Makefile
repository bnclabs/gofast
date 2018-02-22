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
	rm -rf *.svg *.pprof *.mprof
	make -C perf clean
	make -C example clean
