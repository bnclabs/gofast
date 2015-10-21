* add test cases:
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
* replace pktpool with pools, one for each routine.
* try to remove select{} blocks as much as possible.
* support snappy compression.
* document programming model in README page.
* add snappy compression.
* add tools/requests as part of travis-ci.
* run mprof and optimize on escaping variables.
* add testcases to improve code coverage.
* try gofast on raspberry-pi.
