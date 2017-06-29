Getting started with gofast
===========================

Before diving into code let us see some basic ideas that will come together
when we start coding with gofast.

**Message**

All messages exchanged via gofast transport must implement the
[Message](https://godoc.org/github.com/prataprc/gofast#Message)
interface. After initializing the transport and once Handshake() is called
with remote, the transport object can be concurrently shared by several
go-routines, each routine, can initiate a new request with remote and exchange
one or more messages.

**BinMessage**

* All incoming messages are typed as BinMessage{}.
* BinMessage.ID gives uint64 number for message-id.
* BinMessage.Data gives the actual message as serialized bytes.

**RequestCallback**

* Before calling Handshake() with remote node, application
must register a callback for every message-id, with SubscribeMessage.
Each concrete type can be assigned a message-id.
* Applications can also register a catch-all callback with
DefaultHandler().
* For every registered incoming message, RequestHandler will be dispatched
with adequate arguments.
* Handler shall not block and must be as light weight as possible. In
other-words, it should quickly return back to the caller, or else it will
impact the latency and eventually throughput.
* If request is initiating a stream of messages from remote,
handler should return a stream-callback.

**Stream object**

Stream objects are obtained in two ways:

* When a call to transport.Stream() succeeds, a new stream object is returned.
* As part of RequestCallback argument.

Once the request is initiated, all communication with remote must happen
on the stream object, via its `Response()`, `Stream()`, `Close()` methods.
Again, once the request is initiated, either node can send messages to
remote.

Note that POST requests don't involve streams.

**StreamCallback**

Similar to sending messages via stream-objects, incoming messages are
dispatched via stream-callback.

* Node which initiates a stream-request, via `transport.Stream()`, should
supply the StreamCallback as argument.
* Node which is receiving the new request should supply the
StreamCallback as part of the request-handler's return value.
* Similar to RequestCallback, StreamCallback should not block and must be
as light-weight as possible. In other-words, it should quickly return back
to the caller, or else it will impact the latency and eventually throughput.

**IMPORTANT: Both sides must close the stream to release the stream object
back to the transport**

**DefaultSettings**

It is possible to tune gofast transport for latency, throughput, and memory
footprint. If settings argument to NewTransport is passed as nil, transport
will fall back to the default settings. Settings can be configured for each
Transport instance. To learn more about settings parameters
[refer here](https://godoc.org/github.com/prataprc/gofast#DefaultSettings)

Get coding
----------

**Example server-code**

```go
conn, err := lis.Accept()
ver := gofast.Version64(1)
setts := gofast.DefaultSettings()
trans, err := gofast.NewTransport("example-server", conn, &ver, setts)
go func(trans *gf.Transport) {
    trans.FlushPeriod(flushtick * time.Millisecond)
    trans.SendHeartbeat(1 * time.Second)
    trans.SubscribeMessage(
        &msgPost{},
        func(s *gf.Stream, msg gf.BinMessage) gf.StreamCallback {
            // Fill up your handler code here.
            return nil
        })
    trans.Handshake()
}(trans)
```

**Client-code**

```go
conn, err := net.Dial("tcp", serveraddr)
setts := gofast.DefaultSettings()
trans, err := gf.NewTransport("example-client", conn, gofast.Version64, setts)
trans.FlushPeriod(flushtick * time.Millisecond)
trans.SendHeartbeat(1 * time.Second)
trans.SubscribeMessage(&msgPost{}, nil)
trans.Handshake()
```

**To configure range of opaque values**

```go
setts := gofast.DefaultSettings()
setts["opaque.start"] = 1000
setts["opaque.end"] = 10000
trans, err := gf.NewTransport("example", conn, gofast.Version64, setts)
```

This would mean, at any given time, 9000 streams can be active on the
connection.

**To post a message to remote**

```go
msg := &MsgPostDocument{...}
trans.Post(msg, false); err != nil {
```

Note that either node can post a message to the remote node. POST messages
don't respond back.

**To request a response from remote**

```go
req := &MsgGetDocument{...}
resp := &ResponseDocument{}
transc.Request(req, true, resp); err != nil {
fmt.Println(resp)
```

Note that either node can request a response from remote node. For every
REQUEST message remote node will send a single response. There will be
no other exchange for that request.

**To request a stream response from remote**

Streaming protocols have some learning curve. Gofast is no exception. But once
the basic idea is understood peer-to-peer, bi-directional streaming becomes easy.

```go
var resps []*RangeResponse // place to collect all streamed response.
var stream *gofast.Stream
req := &MsgRangeQuery{...}
stream, _ = trans.Stream(req, true, func(bmsg BinMessage, ok bool) {
    if ok {
        resp := &RangeResponse{}
        resp.Decode(bmsg.Data)
        resps = append(resps, resp)
    } else { // remote has closed the stream.
        fmt.Println("received %v entries", len(resps))
    }
    if len(resps) > 100 { // we have received more than 100 entries, close
        stream.Close()
    }
}
```

The last argument to transport.Stream() is StreamCallback that will fire
for every incoming message from remote. Also, note that the local node
closes the stream once it receives 100 messages. When the remote node,
which acts as a server here, receives the close message it can
to stop sending the entry responses, while messages that are already
in-flight will get dropped by the client node.

Remote node might look like:

```go
trans.SubscribeMessage(
    &MsgRangeQuery{},
    func(s *Stream, rxmsg BinMessage) StreamCallback {
        remoteclosed := make(chan struct{})
        go func(stream *Stream, rxmsg BinMessage) {
            for _, entry := range db.Range(getargs(BinMessage)) {
                select {
                case _, ok <- remoteclosed:
                    if ok == false { // remote has closed
                        return
                    }
                default:
                }
                // send entry
                err := stream.Stream(&entry, true)
            }
            stream.Close()
        }(stream, rxmsg)

        return func(bmsg BinMessage, ok bool) {
            if ok == false {
                close(remoteclosed)
            }
        }
    })
```

RequestCallback spawn a go-routine to scan the db. This is because callbacks
in gofast cannot block and should return immediately. Instead of spawning a
go-routine, callback can send a message to worker pools to avoid spawn-kill
overheads.

RequestCallback, in this case, return a StreamCallback function. This is
because server allows remote to cancel the request at any point. And because
server expects only a close signal from remote it does not interpret the
BinMessage argument.

Note that either node can request a stream from remote node. Once
request is initiated with remote, either node can send (aka stream)
messages to remote.
