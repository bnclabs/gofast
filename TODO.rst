* improve code-coverage to 90%.
* golang 1.6 has default support for http 2.0. benchmark gofast with
  http 2.0.
* run travis for go1.4, go1.5, go1.6
* write-up on using `peerversion` to build distributed applications
  supporting rolling upgrade.
* document programming model in README page.
* try gofast on raspberry-pi.
* test case with 1-byte packet.
* test case with 0-byte packet.
* check whether select{} blocks are costly ?
* support snappy compression.
* find a way to add SendHeartbeat() in verify/{client.go,server.go}
* $ client :9900 stream # fails for some reason. make it more resilient.
