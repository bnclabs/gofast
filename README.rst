gofast implements a High performance protocol for distributed
applications with following features,

1. post requests from client to server.
2. server send back a single response for each request from client.
3. bi-directional streaming, where client can initiate a request
   and follow it up with more messages, similarly for a request from
   client server can send a stream of message.
4. bi-directional streaming where either client or server can close
   the stream.
5. concurrent request on a single connection, improves throughput
   when latency is higher.
6. custom encoding and compression can be used,
   - out of the box, gzip and lzw compression are supplied.
   - will soon be adding support for JSON and BSON encoding.

**Protocol framing**

Simple protocol framing.

.. code-block::

    0               8               16              24            31
    +---------------+---------------+---------------+---------------+
    |         Message type          |             Flags             |
    +---------------+---------------+---------------+---------------+
    |                     Opaque value (uint32)                     |
    +---------------+---------------+---------------+---------------+
    |                   payload-length (uint32)                     |
    +---------------+---------------+---------------+---------------+
    |                        payload ....                           |
    +---------------+---------------+---------------+---------------+

    flags

        +---------------+---------------+
    byte|       0       |       1       |
        +---------------+---------------+
    bits|0 1 2 3 4 5 6 7|0 1 2 3 4 5 6 7|
        +-------+-------+---------------+
       0| COMP. |  ENC. |R S E          |
        +-------+-------+---------------+

- mtype-field states the type of `payload` carried by the packet.
- flags-field:
  * `ENC` encoding format.
  * `COMP` compression type.
  * `R` packet is request (client to server) or response (server to client).
    every request initiates a new session.
  * `S` packet is part of streaming messages.
  * `E` end of session.
  * `R`, `S`, `E` are always interpreted in the context of opaque.
- opaque value used for concurrent requests on the same connection.
