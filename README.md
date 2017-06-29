-------
Summary
-------

High performance protocol for distributed applications.

[![Build Status](https://travis-ci.org/prataprc/gofast.png)](https://travis-ci.org/prataprc/gofast)
[![Coverage Status](https://coveralls.io/repos/prataprc/gofast/badge.png?branch=master&service=github)](https://coveralls.io/github/prataprc/gofast?branch=master)
[![GoDoc](https://godoc.org/github.com/prataprc/gofast?status.png)](https://godoc.org/github.com/prataprc/gofast)

Goal
----

* [CBOR](http://cbor.io),(Concise Binary Object Representation) based
  protocol, avoids yet another protocol frame.
* Well formed gofast packets are fully [CBOR](http://cbor.io) compliant.
* Symmetric protocol - like all socket programming client initiates
  the connection, but there after client and server can exchange
  messages like peers, that is both ends can:
  - `POST` messages to remote node.
  - `REQUEST` a `RESPONSE` from remote node.
  - Start one or more bi-direction `STREAM` with remote node.
* Concurrent request on a single connection, improves throughput when
  latency is high.
* Configurable batching of packets scheduled for transmission.
* Periodic flusher for batching response and streams.
* Send periodic heartbeat to remote node.
* Add transport level compression like `gzip`, `lzw` ...
* Sub-μs protocol overhead.
* Scales with number of connection and number of cores.
* And most importantly - does not attempt to solve all the world's problem.

Quick links
-----------

* [Frame-format](#frame-format).
* [Settings](https://godoc.org/github.com/prataprc/gofast#DefaultSettings).
* [Getting-started](docs/gettingstarted.md)
* [Performance benchmark](perf/README.md)
* [How to contribute](#how-to-contribute)

**dev-notes**

* `Transport{}` is safe for concurrent access.
* `Stream{}` is not safe for concurrent access.

Frame-format
------------

Frames encode packet in CBOR compliant form, identifying the exchange
as one of the following:

**Post-request**, client post a packet and expects no response:

```text
| 0xd9 0xd9f7 | 0xc6 | packet |
```

**Request-response**, client make a request and expects a single response:

```text
| 0xd9 0xd9f7 | 0x81 | packet |
```

**Bi-directional streaming**, where client and server will have to close
the stream by sending a 0xff:

```text
     | 0xd9 0xd9f7         | 0x9f | packet1    |
            | 0xd9 0xd9f7  | 0xc7 | packet2    |
            ...
            | 0xd9 0xd9f7  | 0xc8 | end-packet |
```

* `0xd9` says frame is a tag with 2-bye extension.
* Following two bytes `0xd9f7` is tag-number `Tag-55799`.
* As per the RFC - **0xd9 0xd9f7 appears not to be in use as a
  distinguishing mark for frequently used file types**.
* Maximum length of a packet can be 4GB.
* 0xc6 is gofast reserved tag (tagvalue-6) to denote that the following
  packet is a post.
* 0x81 denotes a cbor array of single item, a special meaning for new
  request that expects a single response from remote.
* 0x9f denotes a cbor array of indefinite items, a special meaning
  for a new request that starts a bi-directional stream.
* 0xc7 is gofast reserved tag (tagvalue-7) to denote that the following
  packet is part of a stream.
* 0xc8 is gofast reserved tag (tagvalue-8) to denote that this packet
  is an end-packet closing the bi-directional stream.
* `Packet` shall always be encoded as CBOR byte-array.

Except for post-request, the exchange between client and server is always
symmetrical.

Packet-format
-------------

A packet is CBOR byte-array that can carry tags and payloads, it has
the following format:

```text
  | len | tag1 |         payload1               |
               | tag2 |      payload2           |
                      | tag3 |   payload3       |
                             | tag 4 | hdr-data |
```

* Entire packet is encoded as CBOR byte-array.
* `len` is nothing but the byte-array length (Major-type-2).
* Payload shall always be encoded as CBOR byte-array.
* HDR-DATA shall always be encoded as CBOR map.
* Tags are uint64 numbers that will either be prefixed to payload or hdr-data.
* Tag1, will always be a opaque number falling within a reserved tag-space
  called opaque-space.
* **Opaque-space should not start before 256**.
* Tag2, Tag3 can be one of the values predefined by this library.
* Final embedded tag, in this case tag4, shall always be tagMsg (value 37).

**hdr-data**

* TagId, identifies message with unique id.
* TagData, identified encoded message as byte array.

**end-packet**

```text
    | len | tag1  | 0xff |
```

* If packet denotes a stream-end, payload will be 1-byte 0xff,
  and not encoded as byte-array.

**Reading a frame from socket**

Framing of packets are done such that any gofast packet will at-least be 9
bytes long. Here is how it happens:

* The smallest `payload` should be at-least 1 byte length, because it is
encoded as CBOR byte-array or as end-packet (0xff).
* Every payload will be prefix with opaque-tag, which is always >= 256 in
value. That is 3 bytes.
* Entire `packet` is encoded as CBOR byte-array, that is another 1 byte
overhead.
* And finally framing always takes up 4 bytes.

That is a total of: 1 + 3 + 1 + 4

Incidentally these 9 bytes are enough to learn how many more bytes to read
from the socket to complete the entire packet.

Reserved-tags
-------------

Following list of CBOR tags are reserved for gofast protocol. Some of these
tags may be standardised by CBOR specification, but the choice of these
tag values will speed-up frame encoding and decoding.

`tag-6`
    TagPost, following tagged CBOR byte-array carries a POST request.

`tag-7`
    TagStream, following tagged CBOR byte-array carries a STREAM message.

`tag-8`
    TagFinish, following is tagged CBOR breakstop (0xff) item.

`tag-37`
    TagMsg, following CBOR map carries message header and data.

`tag-38`
    TagId, used as key in CBOR header-data mapping to unique message ID.

`tag-39`
    TagData, used as key in CBOR header-data mapping to message, binary
    encoded as CBOR byte-array.

`tag-40`
    TagGzip, following CBOR byte array is compressed using gzip encoding.

`tag-41`
    TagLzw, following CBOR byte array is compressed using gzip encoding.

These reserved tags are not part of CBOR specification or IANA registry,
please refer/follow issue [#1](https://github.com/prataprc/gofast/issues/1).

Sizing
------

Based on the configuration following heap allocations can affect memory
sizing.

* Batch of packets copied into a single buffers before flushing into socket:
  `writebuf := make([]byte, batchsize*buffersize)`
  for configured range of opaque space between [opaque.start, opaque.end]

* As many stream{} objects will be pre-created and pooled:
  `((opaque.end-opaque.start)+1) * sizeof(stream{})`

* Each stream will allocate 3 buffers for sending/receiving packets.
  `buffersize * 3`

* As many txproto{} objects will be pre-create and pooled:
  `((opaque.end-opaque.start)+1) * sizeof(txproto{})`

* As many tx protocol encode buffers will be pre-created and pooled:
  `((opaque.end-opaque.start)+1) * buffersize`

Http-endpoints
--------------

Gofast exports per transport statistics and aggregated statistics for
debugging and performance tracking. For more details check out the
godoc documentation.

How to contribute
-----------------

* Pick an issue, or create an new issue. Provide adequate documentation for
  the issue.
* Assign the issue or get it assigned.
* Work on the code, once finished, raise a pull request.
* Golog is written in [golang](https://golang.org/), hence expected to follow the
  global guidelines for writing go programs.
