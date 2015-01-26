package gofast

import "testing"
import "time"
import "fmt"
import "log"
import "reflect"

var _ = fmt.Sprintf("dummy")

// clients are started and closed per test case.
var clientConfig = map[string]interface{}{
	"maxPayload":     1024 * 1024,
	"writeDeadline":  4 * 1000, // 4 seconds
	"muxChanSize":    100000,
	"streamChanSize": 10000,
}

// server is common to all test cases.
var serverConfig = map[string]interface{}{
	"maxPayload":     1024 * 1024,
	"writeDeadline":  4 * 1000, // 4 seconds
	"reqChanSize":    1000,
	"streamChanSize": 10000,
}

var flags = TransportFlag(EncodingBinary)
var refCh = make(chan interface{}, 100000)

var testPostReq = 0
var testSimpleReq = 1
var testParSimpleReq1 = 2
var testParSimpleReq2 = 3
var testStreamReq1 = 4
var testStreamReq2 = 5
var tmsgs = []struct {
	mtypeReq  uint16
	requests  []string
	mtypeResp uint16
	responses []string
}{
	{1, // Post
		[]string{"post-requests1"},
		2,
		[]string{"post-response2"},
	},
	{3, // Request
		[]string{"simple-requests1"},
		4,
		[]string{"simple-response2"},
	},
	{5, // Par-Request
		[]string{"par-requests1"},
		6,
		[]string{"par-response1"},
	},
	{7, // Par-Request
		[]string{"par-requests2"},
		8,
		[]string{"par-response2"},
	},
	{9, // streaming-request
		[]string{"strm1-req1", "strm1-req2", "strm1-req3", "strm1-req4"},
		10,
		[]string{"strm1-res1", "strm1-res2", "strm1-res3", "strm1-res4"},
	},
	{11, // streaming-request
		[]string{"strm2-req1", "strm2-req2", "strm2-req3", "strm2-req4"},
		12,
		[]string{"strm2-res1", "strm2-res2", "strm2-res3", "strm2-res4"},
	},
}

var server *Server
var host = ":9999"

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
				if string(request.([]byte)) == req {
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
				if string(request.([]byte)) == req {
					fn(tmsg.mtypeResp, []byte(tmsg.responses[i]), true)
					break
				}
			}
		})
	// setup concurrent-request handler
	server.SetRequestHandlerFor(
		tmsgs[testParSimpleReq1].mtypeReq,
		func(request interface{}, fn ResponseSender) {
			tmsg := tmsgs[testParSimpleReq1]
			if string(request.([]byte)) == tmsg.requests[0] {
				fn(tmsg.mtypeResp, []byte(tmsg.responses[0]), true)
			}
		})
	// setup concurrent-request handler
	server.SetRequestHandlerFor(
		tmsgs[testParSimpleReq2].mtypeReq,
		func(request interface{}, fn ResponseSender) {
			tmsg := tmsgs[testParSimpleReq2]
			if string(request.([]byte)) == tmsg.requests[0] {
				fn(tmsg.mtypeResp, []byte(tmsg.responses[0]), true)
			}
		})
	// setup streaming-request1 handler
	streamch1 := make(chan [3]interface{}, 100)
	server.SetStreamHandlerFor(
		tmsgs[testStreamReq1].mtypeReq,
		StreamHandler(
			func(request interface{}, send ResponseSender) chan [3]interface{} {
				tmsg := tmsgs[testStreamReq1]
				send(tmsg.mtypeResp, []byte(tmsg.responses[0]), false)
				go func() {
					for {
						msg := <-streamch1
						mtype, request, _ := msg[0].(uint16), msg[1], msg[2].(bool)
						if mtype == MtypeEmpty {
							send(tmsg.mtypeResp, nil, true)
						} else if mtype != tmsg.mtypeReq {
							log.Fatalf(
								"expected mtype %v, got %v", tmsg.mtypeReq, mtype)
						} else {
							for i, req := range tmsg.requests {
								if string(request.([]byte)) == req {
									send(tmsg.mtypeResp,
										[]byte(tmsg.responses[i]), false)
									break
								}
							}
						}
					}
				}()
				return streamch1
			}))

	// setup streaming-request2 handler
	streamch2 := make(chan [3]interface{}, 100)
	server.SetStreamHandlerFor(
		tmsgs[testStreamReq2].mtypeReq,
		StreamHandler(
			func(request interface{}, send ResponseSender) chan [3]interface{} {
				tmsg := tmsgs[testStreamReq2]
				send(tmsg.mtypeResp, []byte(tmsg.responses[0]), false)
				go func() {
					for {
						msg := <-streamch2
						mtype, request := msg[0].(uint16), msg[1]
						finish := msg[2].(bool)
						if mtype == MtypeEmpty {
							send(tmsg.mtypeResp, nil, true)
						} else if mtype != tmsg.mtypeReq {
							log.Fatalf(
								"expected mtype %v, got %v", tmsg.mtypeReq, mtype)
						} else if finish == true {
							log.Fatalf("did not expect client to close")
						} else {
							for i, req := range tmsg.requests {
								if string(request.([]byte)) == req {
									if i == len(tmsg.requests)-2 {
										send(tmsg.mtypeResp,
											[]byte(tmsg.responses[i]), true)
									} else {
										send(tmsg.mtypeResp,
											[]byte(tmsg.responses[i]), false)
									}
									break
								}
							}
						}
					}
				}()
				return streamch2
			}))
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
		r, err := client.Request(flags, tmsg.mtypeReq, req)
		if err != nil {
			t.Fatal(err)
		}
		response := string(r.([]byte))
		if resp := tmsg.responses[i]; !reflect.DeepEqual(resp, response) {
			t.Fatalf("expected %v got %v", resp, response)
		}
	}

	time.Sleep(100 * time.Millisecond)
	client.Close()
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
				r, err := client.Request(flags, tmsg.mtypeReq, req)
				if err != nil {
					t.Fatal(err)
				}
				resp, response := tmsg.responses[i], string(r.([]byte))
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
				r, err := client.Request(flags, tmsg.mtypeReq, req)
				if err != nil {
					t.Fatal(err)
				}
				resp, response := tmsg.responses[i], string(r.([]byte))
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

func TestStreaming(t *testing.T) {
	LogIgnore()
	//SetLogLevel(LogLevelTrace)

	// make client
	client, err := NewClient(host, clientConfig, nil)
	if err != nil {
		t.Fatal(err)
	}
	client.SetEncoder(EncodingBinary, nil).Start()

	donech := make(chan bool, 10)

	// bidirectional streaming and close at client side.
	var opaque uint32
	responses := make([]string, 0)
	tmsg := tmsgs[testStreamReq1]
	req := tmsg.requests[0] // first request
	opaque, err = client.RequestStream(
		flags, tmsg.mtypeReq, []byte(req),
		func(mtype uint16, opq uint32, response interface{}, finish bool) {
			if mtype != tmsg.mtypeResp {
				t.Fatalf("expected mtype %v, got %v\n", tmsg.mtypeResp, mtype)
			} else if opq != opaque {
				t.Fatalf("expected opaque %v, got %v\n", opaque, opq)
			} else if finish == true {
				donech <- true // additional done
			} else {
				responses = append(responses, string(response.([]byte)))
			}
			donech <- true
		})
	if err != nil {
		t.Fatal(err)
	}
	err = client.Stream(opaque, tmsg.mtypeReq, []byte(tmsg.requests[1]), false)
	if err != nil {
		t.Fatal(err)
	}
	err = client.Stream(opaque, tmsg.mtypeReq, []byte(tmsg.requests[2]), false)
	if err != nil {
		t.Fatal(err)
	}
	err = client.StreamClose(opaque)
	if err != nil {
		t.Fatal(err)
	}
	<-donech
	<-donech
	<-donech
	<-donech
	if !reflect.DeepEqual(tmsg.responses[:len(tmsg.responses)-1], responses) {
		t.Fatalf("expected %v got %v", tmsg.responses, responses)
	}

	// bidirectional streaming and close at server side.
	responses = make([]string, 0)
	tmsg = tmsgs[testStreamReq2]
	req = tmsg.requests[0] // first request
	opaque, err = client.RequestStream(
		flags, tmsg.mtypeReq, []byte(req),
		func(mtype uint16, opq uint32, response interface{}, finish bool) {
			if mtype != tmsg.mtypeResp {
				t.Fatalf("expected mtype %v, got %v\n", tmsg.mtypeResp, mtype)
			} else if opq != opaque {
				t.Fatalf("expected opaque %v, got %v\n", opaque, opq)
			} else if finish == true {
				donech <- true
			}
			responses = append(responses, string(response.([]byte)))
			donech <- true
		})
	if err != nil {
		t.Fatal(err)
	}
	for _, req := range tmsg.requests[1:len(tmsg.requests)] {
		err = client.Stream(opaque, tmsg.mtypeReq, []byte(req), false)
		if err != nil {
			t.Fatal(err)
		}
	}
	<-donech
	<-donech
	<-donech
	<-donech
	<-donech
	if !reflect.DeepEqual(tmsg.responses[:len(tmsg.responses)-1], responses) {
		t.Fatalf("expected %v got %v", tmsg.responses, responses)
	}

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
