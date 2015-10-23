* add testcases to improve code coverage.
  * test case for cbor.go:valtext2cbor().
  * rx.go:test receiving start-stream.
  * rx.go:test receiving Close() stream.
  * rx.go:test receiving msg for on going stream.
  * rx.go:getch().
  * rx.go:test receiving gzipped messages.
  * stream.go:Stream().
  * stream.go:Close().
  * transport.go:Stream().
  * tx.go:test sending gzipped messages.
  * util.go:bytes2str().
* tools/posts similar to tools/requests
* tools/streams similar to tools/requests
* add tools/{posts,requests,streams} to travis-ci.
* document programming model in README page.
* run mprof and optimize on escaping variables.
* check whether select{} blocks are costly ?
* support snappy compression.
* try gofast on raspberry-pi.
