package gofast

import "testing"
import "time"
import "fmt"
import "log"
import "reflect"

var clientConfig = map[string]interface{}{
    "maxPayload":     1024 * 1024,
    "writeDeadline":  4 * 1000, // 4 seconds
    "muxChanSize":    100000,
    "streamChanSize": 10000,
}
var serverConfig = map[string]interface{}{
    "maxPayload":     1024 * 1024,
    "writeDeadline":  4 * 1000, // 4 seconds
    "reqChanSize":    1000,
    "streamChanSize": 10000,
}

type refMsg []interface{}

var flags = TransportFlag(EncodingBinary)
var server *Server
var refCh = make(chan []interface{}, 100000)
var host = ":9999"
var _ = fmt.Sprintf("dummy")

var tmsgs = []struct{
    mtypeReq   uint16
    requests  [][]byte
    mtypeResp uint16
    response  [][]byte
}{
 { 0x0001, // Post
   [][]byte{[]byte("post-requests1")},
   0x0002,
   [][]byte{[]byte("post-response2")},
 },
 { 0x0003, // Request
   [][]byte{[]byte("simple-requests1")},
   0x0004,
   [][]byte{[]byte("simple-response2")},
 },
 { 0x0005, // Par-Request
   [][]byte{[]byte("par-requests1")},
   0x0006,
   [][]byte{[]byte("par-response2")},
 },
 { 0x0007, // Par-Request
   [][]byte{[]byte("par-requests3")},
   0x0008,
   [][]byte{[]byte("par-response4")},
 },
}
var testPostReq = 0
var testSimpleReq = 1
var testParSimpleReq1 = 2
var testParSimpleReq2 = 3

func init() {
    var err error
    server, err = NewServer(host, serverConfig, nil)
    if err != nil {
        log.Fatal(err)
    }
    server.SetEncoder(EncodingBinary, nil)

    tmsg := tmsgs[testPostReq]
    server.SetPostHandler(
        func (request interface{}) {
            refCh <- []interface{}{request}
        })

    tmsg = tmsgs[testSimpleReq]
    server.SetRequestHandler(
        func(req interface{}, fn ResponseSender) {
            if reflect.DeepEqual(req, tmsg.requests[0]) {
                fn(tmsg.mtypeResp, tmsg.responses[0], true /*finish*/)
            }
        })

    tmsg = tmsgs[testParSimpleReq1]
    server.SetRequestHandlerFor(
        tmsg.mtypeReq,
        func(req interface{}, fn ResponseSender) {
            if reflect.DeepEqual(req, tmsg.requests[0]) {
                fn(tmsg.mtypeResp, tmsg.responses[0], true /*finish*/)
            }
        })

    tmsg = tmsgs[testParSimpleReq2]
    server.SetRequestHandlerFor(
        tmsg.mtypeReq,
        func(req interface{}, fn ResponseSender) {
            if reflect.DeepEqual(req, tmsg.requests[0]) {
                fn(tmsg.mtypeResp, tmsg.responses[0], true /*finish*/)
            }
        })
}

func TestPost(t *testing.T) {
    LogIgnore()
    //SetLogLevel(LogLevelTrace)

    // make client
    client, err := NewClient(host, clientConfig, nil)
    if err != nil {
        t.Fatal(err)
    }
    client.SetEncoder(EncodingBinary, nil).Start()
    tmsg := tmsgs[testPostReq]
    if err := client.Post(flags, tmsg.mtype, postPayload); err != nil {
        t.Fatal(err)
    }
    validateRefCh([]refMsg{ refMsg{1, postPayload} }, t)
    client.Close()
    time.Sleep(100 * time.Millisecond)
}

func TestRequest(t *testing.T) {
    LogIgnore()
    //SetLogLevel(LogLevelTrace)

    // make client
    client, err := NewClient(host, clientConfig, nil)
    if err != nil {
        t.Fatal(err)
    }
    client.SetEncoder(EncodingBinary, nil).Start()

    validateRequest(client, []byte("request1"), []byte("response1"), t)
    validateRequest(client, []byte("request2"), []byte("response2"), t)

    client.Close()
    time.Sleep(100 * time.Millisecond)
}

func TestRequestWith(t *testing.T) {
    //LogIgnore()
    SetLogLevel(LogLevelTrace)
    // make client
    client, err := NewClient(host, clientConfig, nil)
    if err != nil {
        t.Fatal(err)
    }
    client.SetEncoder(EncodingBinary, nil).Start()

    req1, res1 := []byte("request1"), []byte("response1")
    req2, res2 := []byte("request1"), []byte("response1")
    go func() {
        for i := 0; i < 1000; i++ {
            validateRequestWith(client, req1, res1, t)
        }
    }()
    go func() {
        for i := 0; i < 1000; i++ {
            validateRequestWith(client, req2, res2, t)
        }
    }()
    for i := 2000; i > 0; i-- {
        <-refCh
    }

    time.Sleep(100 * time.Millisecond)
    client.Close()
}

func BenchmarkPostTx(b *testing.B) {
    // make client
    client, err := NewClient(host, clientConfig, nil)
    if err != nil {
        b.Fatal(err)
    }
    client.SetEncoder(EncodingBinary, nil)
    client.Start()

    for i := 0; i < b.N; i++ {
        client.Post(flags, 0x0001 /*mtype*/, postPayload)
        b.SetBytes(int64(len(postPayload)))
    }
    refCh = make(chan []interface{}, 100000)
    client.Close()
}

func validateRefCh(refmsgs []refMsg, tb testing.TB) {
    if len(refCh) != len(refmsgs) {
        tb.Fatal("unexpected no. of msgs in refCh %v\n", len(refCh))
    }
    srvmsgs := make([]refMsg, 0, len(refCh))
    for len(refCh) > 0 {
        srvmsgs = append(srvmsgs, <-refCh)
    }
    for i, refmsg := range refmsgs {
        if !reflect.DeepEqual(refmsg, srvmsgs[i]) {
            tb.Fatal("expected %v got %v", refmsg, srvmsgs[i])
        }
    }
}

func validateRequest(client *Client, req, resp interface{}, tb testing.TB) {
    response, err := client.Request(flags, 0x0001 /*mtype*/, req)
    if err != nil {
        tb.Fatal(err)
    } else if !reflect.DeepEqual(resp, response) {
        tb.Fatalf("mismatch in response %T %s", response, response)
    }
    tb.Logf("%v done", string(req.([]byte)))
}

func validateRequestWith(
    client *Client, req, resp interface{}, tb testing.TB) {

    response, err := client.RequestWith(0x0001 /*mtype*/, flags, req)
    if err != nil {
        tb.Fatal(err)
    } else if !reflect.DeepEqual(resp, response) {
        tb.Fatalf("mismatch in response %T %s", response, response)
    }
    tb.Logf("%v done", string(req.([]byte)))
}
