build:
	go build

test:
	go test -v -race -timeout 4000s -test.run=. -test.bench=. -test.benchmem=true

coverage:
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm -rf coverage.out

clean:
	rm -f tools/client/{client,client.mprof,client.pprof,*.svg}
	rm -f tools/server/{server,server.mprof,server.pprof,*.svg}
	rm -f tools/{*.mprof,*.pprof,*.svg}
