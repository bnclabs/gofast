based on the configuration following heap allocations can affect memory
sizing.

batch of packets copied into a single buffers before flushing into socket:

    tcpwrite_buf := make([]byte, batchsize*buffersize)

for configured range of opaque space between [opaque.start, opaque.end]

* as many stream{} objects will be pre-created and pooled:
  ``((opaque.end-opaque.start)+1) * sizeof(stream{})``
* each stream will allocate 3 buffers for sending/receiving packets.
  ``buffersize * 3``
* as many txproto{} objects will be pre-create and pooled:
  ``((opaque.end-opaque.start)+1) * sizeof(txproto{})``
* as many tx protocol encode buffers will be pre-created and pooled:
  ``((opaque.end-opaque.start)+1) * buffersize``
