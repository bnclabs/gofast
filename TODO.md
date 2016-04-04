* review panic calls, add local-addr and remote-addr as part of
  panic call.
* write-up on using `peerversion` to build distributed applications
  supporting rolling upgrade.
* default configuration.
* have a fixed tag name for POST messages. Avoid acquiring a stream.
* verify takes > 5GB of memory, investigate the cause.
* enable `-race` while verifying.
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
