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

var flags = TransportFlag(EncodingBinary)
var server *Server
var refCh = make(chan interface{}, 100000)
var host = ":9999"
var _ = fmt.Sprintf("dummy")

var tmsgs = []struct {
	mtypeReq  uint16
	requests  [][]byte
	mtypeResp uint16
	responses [][]byte
}{
	{1, // Post
		[][]byte{[]byte("post-requests1")},
		2,
		[][]byte{[]byte("post-response2")},
	},
	{3, // Request
		[][]byte{[]byte("simple-requests1")},
		4,
		[][]byte{[]byte("simple-response2")},
	},
	{5, // Par-Request
		[][]byte{[]byte("par-requests1")},
		6,
		[][]byte{[]byte("par-response1")},
	},
	{7, // Par-Request
		[][]byte{[]byte("par-requests2")},
		8,
		[][]byte{[]byte("par-response2")},
	},
}
var testPostReq = 0
var testSimpleReq = 1
var testParSimpleReq1 = 2
var testParSimpleReq2 = 3

func init() {
	var err error
	// start a global server for this test cases.
	server, err = NewServer(host, serverConfig, nil)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	server.SetEncoder(EncodingBinary, nil)
	// setup post handler
	server.SetPostHandler(
		func(request interface{}) {
			for i, req := range tmsgs[testPostReq].requests {
				if reflect.DeepEqual(request, req) {
					refCh <- tmsgs[testPostReq].responses[i]
					break
				}
			}
		})
	// setup request handler
	server.SetRequestHandler(
		func(request interface{}, fn ResponseSender) {
			tmsg := tmsgs[testSimpleReq]
			for i, req := range tmsg.requests {
				if reflect.DeepEqual(request, req) {
					fn(tmsg.mtypeResp, tmsg.responses[i], true)
					break
				}
			}
		})
	// setup concurrent-request handler
	server.SetRequestHandlerFor(
		tmsgs[testParSimpleReq1].mtypeReq,
		func(req interface{}, fn ResponseSender) {
			tmsg := tmsgs[testParSimpleReq1]
			if reflect.DeepEqual(req, tmsg.requests[0]) {
				fn(tmsg.mtypeResp, tmsg.responses[0], true)
			}
		})
	// setup concurrent-request handler
	server.SetRequestHandlerFor(
		tmsgs[testParSimpleReq2].mtypeReq,
		func(req interface{}, fn ResponseSender) {
			tmsg := tmsgs[testParSimpleReq2]
			if reflect.DeepEqual(req, tmsg.requests[0]) {
				fn(tmsg.mtypeResp, tmsg.responses[0], true)
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
	for i, req := range tmsg.requests {
		err := client.Post(flags, tmsg.mtypeReq, req)
		if err != nil {
			t.Fatal(err)
		}
		validateRefCh([]interface{}{tmsg.responses[i]}, t)
	}
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

	tmsg := tmsgs[testSimpleReq]
	for i, req := range tmsg.requests {
		response, err := client.Request(flags, tmsg.mtypeReq, req)
		if err != nil {
			t.Fatal(err)
		}
		if resp := tmsg.responses[i]; !reflect.DeepEqual(resp, response) {
			t.Fatalf("expected %v got %v", resp, response)
		}
	}

	client.Close()
	time.Sleep(100 * time.Millisecond)
}

func TestRequestConcur(t *testing.T) {
	LogIgnore()
	//SetLogLevel(LogLevelTrace)

	// make client
	client, err := NewClient(host, clientConfig, nil)
	if err != nil {
		t.Fatal(err)
	}
	client.SetEncoder(EncodingBinary, nil).Start()

	done := make(chan bool, 2)

	go func() {
		tmsg := tmsgs[testParSimpleReq1]
		for j := 0; j < 100; j++ {
			for i, req := range tmsg.requests {
				response, err := client.Request(flags, tmsg.mtypeReq, req)
				if err != nil {
					t.Fatal(err)
				}
				resp := tmsg.responses[i]
				if !reflect.DeepEqual(resp, response) {
					t.Fatalf("expected %v got %v", resp, response)
				}
			}
		}
		done <- true
	}()

	go func() {
		tmsg := tmsgs[testParSimpleReq2]
		for j := 0; j < 100; j++ {
			for i, req := range tmsg.requests {
				response, err := client.Request(flags, tmsg.mtypeReq, req)
				if err != nil {
					t.Fatal(err)
				}
				resp := tmsg.responses[i]
				if !reflect.DeepEqual(resp, response) {
					t.Fatalf("expected %v got %v", resp, response)
				}
			}
		}
		done <- true
	}()

	<-done
	<-done
	client.Close()
	time.Sleep(100 * time.Millisecond)
}

func BenchmarkPostTx(b *testing.B) {
	// make client
	client, err := NewClient(host, clientConfig, nil)
	if err != nil {
		b.Fatal(err)
	}
	client.SetEncoder(EncodingBinary, nil)
	client.Start()

	tmsg := tmsgs[testPostReq]
	for i := 0; i < b.N; i++ {
		client.Post(flags, MtypeBinaryPayload, tmsg.requests[0])
		b.SetBytes(int64(len(tmsg.requests[0])))
	}
	refCh = make(chan interface{}, 100000)
	client.Close()
}

func validateRefCh(refmsgs []interface{}, tb testing.TB) {
	srvmsgs := make([]interface{}, 0, len(refCh))
	for i := 0; i < len(refmsgs); i++ {
		srvmsgs = append(srvmsgs, <-refCh)
	}
	if len(srvmsgs) != len(refmsgs) {
		tb.Fatalf(
			"unexpected no. of msgs in refCh %v (%v)\n",
			len(srvmsgs), len(refmsgs))
	}
	for i, refmsg := range refmsgs {
		if !reflect.DeepEqual(refmsg, srvmsgs[i]) {
			tb.Fatalf("expected %v got %v", refmsg, srvmsgs[i])
		}
	}
}
