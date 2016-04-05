Gofast
------

High performance protocol, accompanied by a programming model, for distributed
applications.

|buildstatus| |coveragestatus| |godoc|

Aims at following goal **under development**.

* CBOR_,(Concise Binary Object Representation) based protocol.
  framing, avoid yet another protocol frame.
* well formed gofast packets are fully CBOR_ compliant.
* symmetic protocol - like all socket programming client initiates
  the connection, but there after client and server can exchange
  messages like peers, that is both nodes can:

  * ``POST`` messages to remote node.
  * ``REQUEST`` a ``RESPONSE`` from remote node.
  * start one or more bi-direction ``STREAM`` with remote node.

* concurrent request on a single connection, improves throughput
  when latency is high.
* configurable batching of packets scheduled for transmission.
* periodic flusher for batching response and streams.
* send periodic heartbeat to remote node.
* add transport level compression like ``gzip``, ``lzw`` ...
* sub-μs protocol overhead.
* scales with number of connection and number of cores.
* and most importantly - does not attempt to solve all the
  world's problem.

**dev-notes:**

* ``Transport{}`` is safe for concurrent access.
* ``Stream{}`` is not safe for concurrent access.

**frame-format**

A frame is encoded as finite length CBOR_ map with predefined list
of keys, for example, "id", "data" etc... keys are typically encoded
as numbers so that they can be efficiently packed. This implies that
each of the predefined keys shall be assigned a unique number.

an exchange shall be initiated either by client or server,
exchange can be one of the following.

post-request, client post a packet and expects no response,

.. code-block:: text

     | 0xd9 0xd9f7 | 0xc6 | packet |

request-response, client make a request and expects a single response,

.. code-block:: text

     | 0xd9 0xd9f7 | 0x81 | packet |

bi-directional streaming, where client and server will have to close
the stream by sending a 0xff,

.. code-block:: text

     | 0xd9 0xd9f7         | 0x9f | packet1    |
            | 0xd9 0xd9f7  | 0xc7 | packet2    |
            ...
            | 0xd9 0xd9f7  | 0xc8 | end-packet |

* `packet` shall always be encoded as CBOR_ byte-array.
* the maximum length of a packet can be 4GB.
* 0xc6 is gofast reserved tag (tagvalue-6) to denote that the following
  packet is a post.
* 0x81 denotes a cbor array of single item, a special meaning for new
  request that expects a single response from peer.
* 0x9f denotes a cbor array of indefinite items, a special meaning
  for a new request that starts a bi-directional stream.
* 0xc7 is gofast reserved tag (tagvalue-7) to denote that the following
  package is part of a stream.
* 0xc8 is gofast reserved tag (tagvalue-8) to denote that this packet
  is a end-packet closing the bi-directional stream.

except for post-request, the exchange between client and server is always
symmetrical.

**packet-format**

a single block of binary blob in CBOR_ format, transmitted
from client to server or server to client,

.. code-block:: text

  | len | tag1 |         payload1               |
               | tag2 |      payload2           |
                      | tag3 |   payload3       |
                             | tag 4 | hdr-data |

* payload shall always be encoded as CBOR_ byte-array.
* hdr-data shall always be encoded as CBOR_ map.
* tags are uint64 numbers that will either be prefixed
  to payload or hdr-data.
* tag1, will always be a opaque number falling within a
  reserved tag-space called opaque-space.
* tag2, tag3 can be one of the values predefined by this
  library.
* the final embedded tag, in this case tag4, shall always
  be tagMsg (value 37).

**end-of-stream:**

.. code-block:: text

    | tag1  | 0xff |

* if packet denotes a stream-end, payload will be 1-byte 0xff,
  and not encoded as byte-array.

**useful links:**

* `transport settings <docs/settings.rst>`_
* `transport statistics <docs/statistics.rst>`_

.. _CBOR: http://cbor.io/

.. |buildstatus| image:: https://travis-ci.org/prataprc/gofast.png
    :target: https://travis-ci.org/prataprc/gofast
.. |coveragestatus| image:: https://coveralls.io/repos/prataprc/gofast/badge.png?branch=master&service=github
    :target: https://coveralls.io/github/prataprc/gofast?branch=master
.. |godoc| image:: https://godoc.org/github.com/prataprc/gofast?status.png
    :target: https://godoc.org/github.com/prataprc/gofast

