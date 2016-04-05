Transport statistics
--------------------

all statistics applicable only on the local node.

`n_tx`
    number of messages transmitted.

`n_flushes`
    number of times message-batches where flushed.

`n_txbyte`
    number of bytes transmitted on socket.

`n_txpost`
    number of post messages transmitted.

`n_txreq`
    number of request messages transmitted.

`n_txresp`
    number of response messages transmitted.

`n_txstart`
    number of start messages transmitted, indicates the number of
    streams started by this local node.

`n_txstream`
    number of stream messages transmitted.

`n_txfin`
    number of finish messages transmitted, indicates the number of
    streams closed by this local node, should always match `n_txstart`.

`n_rx`
    number of packets received.

`n_rxbyte`
    number of bytes received from socket.

`n_rxpost`
    number of post messages received.

`n_rxreq`
    number of request messages received.

`n_rxresp`
    number of response messages received.

`n_rxstart`
    number of start messages received, indicates the number of
    streams started by the remote node.

`n_rxstream`
    number of stream messages received.

`n_rxfin`
    number of finish messages received, indicates the number
    of streams closed by the remote node, should always match
    `n_rxstart`

`n_rxbeats`
    number of heartbeats received.

`n_dropped`
    bytes dropped.

`n_mdrops`
    messages dropped.
