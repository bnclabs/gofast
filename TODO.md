* add testcases to improve code coverage.
  * rx.go:test receiving gzipped messages.
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
