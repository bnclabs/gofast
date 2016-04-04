* fix variance and sd computation.
    var:-4h19m32.880533093s sd:-2562047h47m16.854775808s
* sometimes the test case throw the following error.
    127.0.0.1:9117<->127.0.0.1:51824] reading prefix: 4,unexpected EOF
* review panic calls.
* default configuration.
* verify takes > 5GB of memory, investigate the cause.
* opaque-space for a single transport (connection) should be exclusive
  between client and server.
* rename newstream() to newrxstream(), create a complementing function.
* make perf/client.go random to randomly close the stream
  at either end.
* refactor verify and enable random as well.
* test case with 1-byte packet.
* test case with 0-byte packet.
* logs are commented, wrap them under log flag.
* document reserved tags.
* add tools/{posts,requests,streams} to travis-ci.
* document programming model in README page.
* run mprof and optimize on escaping variables.
* check whether select{} blocks are costly ?
* support snappy compression.
* try gofast on raspberry-pi.
- run travis for go1.4, go1.5, go1.6
