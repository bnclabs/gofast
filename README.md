gofast implements a High performance protocol, accompanied by a 
programming model, for distributed applications.

[![Build Status](https://travis-ci.org/prataprc/gofast.png)](https://travis-ci.org/prataprc/gofast)
[![Coverage Status](https://coveralls.io/repos/prataprc/gofast/badge.png?branch=master&service=github)](https://coveralls.io/github/prataprc/gofast?branch=master)
[![GoDoc](https://godoc.org/github.com/prataprc/gofast?status.png)](https://godoc.org/github.com/prataprc/gofast)

* [CBOR, Concise Binary Object Representation](http://cbor.io/) based protocol
  framing, avoid yet another protocol frame.
* well formed gofast packets are fully CBOR compliant.
* symmetic protocol - like all socket programming client initiates
  the connection, but there after client and server can exchange
  messages like peers, that is both nodes can:
  * ``POST`` messages to remote node.
  * ``REQUEST`` a ``RESPONSE`` from remote node.
  * start a bi-direction ``STREAM`` with remote node.
* concurrent request on a single connection, improves throughput
  when latency is high.
* configurable batching of packets scheduled for transmission.
* add transport level compression like `gzip`, `lzw` ...
* ~50K ping/pong exchange on a single connection.
* sub-μs protocol overhead.
* scales with number of connection and number of cores.
* and most importantly - does not attempt to solve all the
  world's problem.

**frame-format**

A frame is encoded as finite length CBOR map with predefined list
of keys, for example, "id", "data" etc... keys are typically encoded
as numbers so that they can be efficiently packed. This implies that
each of the predefined keys shall be assigned a unique number.

an exchange shall be initiated either by client or server,
exchange can be one of the following.

post-request, client post a packet and expects no response:

```text
     | 0xd9 0xd9f7 | packet |
```

request-response, client make a request and expects a single response:

```text

     | 0xd9 0xd9f7 | 0x91 | packet |
```

bi-directional streaming, where client and server will have to close
the stream by sending a 0xff:

```text

     | 0xd9 0xd9f7  | 0x9f | packet1    |
            | 0xd9 0xd9f7  | packet2    |
            ...
            | 0xd9 0xd9f7  | end-packet |
```

* `packet` shall always be encoded as CBOR byte-array of info-type,
  Info26 (4-byte) length.
* 0x91 denotes an array of single item, a special meaning for new
  request that expects a single response from peer.
* 0x9f denotes an array of indefinite items, a special meaning
  for a new stream that starts a bi-directional exchange.

except for post-request, the exchange between client and server is always
symmetrical.

**packet-format**

a single block of binary blob in CBOR format, transmitted
from client to server or server to client:

```text
  | tag1 |         payload1               |
         | tag2 |      payload2           |
                | tag3 |   payload3       |
                       | tag 4 | hdr-data |
```

* payload shall always be encoded as CBOR byte-array.
* hdr-data shall always be encoded as CBOR map.
* tags are uint64 numbers that will either be prefixed
 to payload or hdr-data.
* tag1, will always be a opaque number falling within a
 reserved tag-space called opaque-space.
* tag2, tag3 can be one of the values predefined by this
 library.
* the final embedded tag, in this case tag4, shall always
 be tagMsg (value 37).

end-of-stream:

```text
 | tag1  | 0xff |
```

* if packet denotes a stream-end, payload will be
 1-byte 0xff, and not encoded as byte-array.


**configurations**

 "name"         - give a name for the transport.
 "buffersize"   - maximum size that a packet will need.
 "batchsize"    - number of packets to batch before writing to socket.
 "chansize"     - channel size to use for internal go-routines.
 "tags"         - comma separated list of tags to apply, in specified order.
 "opaque.start" - starting opaque range, inclusive.
 "opaque.end"   - ending opaque range, inclusive.
 "log.level"    - log level to use for DefaultLogger
 "log.file"     - log file to use for DefaultLogger, if empty stdout is used.
 "gzip.level"   - gzip compression level, if `tags` contain "gzip".

**transport statistics**

 n_tx       - number of packets transmitted
 n_flushes  - number of times message-batches where flushed
 n_txbyte   - number of bytes transmitted on socket
 n_txpost   - number of post messages transmitted
 n_txreq    - number of request messages transmitted
 n_txresp   - number of response messages transmitted
 n_txstart  - number of start messages transmitted
 n_txstream - number of stream messages transmitted
 n_txfin    - number of finish messages transmitted
 n_rx       - number of packets received
 n_rxbyte   - number of bytes received from socket
 n_rxpost   - number of post messages received
 n_rxreq    - number of request messages received
 n_rxresp   - number of response messages received
 n_rxstart  - number of start messages received
 n_rxstream - number of stream messages received
 n_rxfin    - number of finish messages received
 n_rxbeats  - number of heartbeats received
 n_dropped  - number of dropped packets
