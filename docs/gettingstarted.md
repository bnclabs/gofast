Getting started with gofast
===========================

Before diving into code let us see some basic ideas that will come together
when we start coding with gofast.

**Message**

All messages exchanged via gofast transport must implement the
[Message](https://godoc.org/github.com/prataprc/gofast#Message)
interface. After initializing the transport and once Handshake() is called
with remote, the transport object can be concurrently shared by several
go-routines, each routine, initiating a new request with remote and exchange
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
otherwords, it should quickly return back to the caller. Otherwise it will
impact the latency and eventually throughput.
* If request is initiating a stream of messages from remote,
handler should return a stream-callback.

**Stream object**

Stream objects are obtained in two ways:

* When a call to transport.Stream() succeeds, a new stream object is returned.
* As part of RequestCallback argument.

Once the request is initiated, all communication with the remote must happen
on the stream object - `Response()`, `Stream()`, `Close()`. Again, once the
request is initiated, either node can send messages to the remote.

Note that POST requests doesn't involve streams.

**StreamCallback**

Note that only new requests (be it POST or REQUEST-RESPONSE
or STREAM-REQUEST) are received via RequestCallback. After the request
is initiated, to send messages to remote, stream-object is
used. And incoming messages are dispatched, for the same request, via
stream-callback.

* Node which initiates a stream-request, via `transport.Stream()`, should
supply the StreamCallback as argument.
* Node which is receiving the new request should supply the
StreamCallback as part of the request-handler's return value.

**DefaultSettings**

It is possible to tune gofast transport for latency, throughput, and memory
footprint. If settings argument to NewTransport is passed as nil, transport
will fall back to the default settings. Settings can be configured for each
Transport instance. To learn more about settings parameters
[refer here](https://godoc.org/github.com/prataprc/gofast#DefaultSettings)

Get coding
----------

```go
conn, err := lis.Accept()
ver := gofast.Version64(1)
setts := gofast.DefaultSettings()
trans, err := gofast.NewTransport("example-server", conn, &ver, setts)
go func(trans *gf.Transport) {
    trans.FlushPeriod(options.flushtick * time.Millisecond)
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

**client-code**

```go
conn, err := net.Dial("tcp", options.addr)
setts := gofast.DefaultSettings()
trans, err := gf.NewTransport("example-client", conn, gofast.Version64, setts)
trans.FlushPeriod(options.flushtick * time.Millisecond)
trans.SendHeartbeat(1 * time.Second)
trans.SubscribeMessage(&msgPost{}, nil)
trans.Handshake()
```

**to configure the range of opaque values**

```go
setts := gofast.DefaultSettings()
setts["opaque.start"] = 1000
setts["opaque.end"] = 10000
trans, err := gf.NewTransport("example", conn, gofast.Version64, setts)
```

This would mean, at any given time, 9000 streams can be active on the
connection.

**to post a message to remote**

```go
msg := &MsgPostDocument{...}
trans.Post(msg, false); err != nil {
```

Note that either node can post a message to the remote node. For POST messages
remote node won't respond back.

**to request a response from remote**

```go
req := &MsgGetDocument{...}
resp := &ResponseDocument{}
transc.Request(req, true, resp); err != nil {
fmt.Println(resp)
```

Note that either node can request a response from remote node. For every
REQUEST message remote node will send a single response. There will be
no other exchange for that request.

**to request a stream response from remote**

```go
var resps []*RangeResponse // collect all entries
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

remote node might look like:

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

Note that either node can request a stream from remote node. Once
request is initiated with remote, either node can send (aka stream)
messages to remote.
