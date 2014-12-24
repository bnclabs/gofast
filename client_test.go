package gofast

import "testing"
import "time"
import "bytes"
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
var postPayload = []byte("hello world")
var server *Server
var host = ":9999"
var _ = fmt.Sprintf("dummy")

func init() {
	var err error
	server, err = NewServer(host, serverConfig, nil)
	if err != nil {
		log.Fatal(err)
	}
	server.SetEncoder(EncodingBinary, nil)
	server.SetPostHandler(postHandler)
}

func TestPost(t *testing.T) {
	LogIgnore()
	//SetLogLevel(LogLevelTrace)

	// make client
	client, err := NewClient(host, clientConfig, nil)
	if err != nil {
		t.Fatal(err)
	}
	client.SetEncoder(EncodingBinary, nil)
	client.Start()

	if err := client.Post(flags, postPayload); err != nil {
		t.Fatal(err)
	}

	client.Close()
	time.Sleep(100 * time.Millisecond)
}

func TestRequest(t *testing.T) {
	LogIgnore()
	//SetLogLevel(LogLevelTrace)

	server.SetRequestHandler(
		func(opaque uint32, req interface{}, respch chan []interface{}) {
			if opaque == 1 {
				respch <- []interface{}{opaque, []byte("response1")}
			}
			if opaque == 2 {
				respch <- []interface{}{opaque, []byte("response2")}
			}
		})

	// make client
	client, err := NewClient(host, clientConfig, nil)
	if err != nil {
		t.Fatal(err)
	}
	client.SetEncoder(EncodingBinary, nil)
	client.Start()

	do := func(req, resp interface{}) {
		response, err := client.Request(flags, req)
		if err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(resp, response) {
			t.Fatalf("mismatch in response %T %s", response, response)
		}
		t.Logf("%v done", string(req.([]byte)))
	}
	do([]byte("request1"), []byte("response1"))
	do([]byte("request2"), []byte("response2"))

	client.Close()
	time.Sleep(100 * time.Millisecond)
}

func TestRequestWith(t *testing.T) {
	//LogIgnore()
	SetLogLevel(LogLevelTrace)

	donech := make(chan bool, 100000)

	server.SetRequestHandlerFor(
		0x80000000,
		func(opaque uint32, req interface{}, respch chan []interface{}) {
			respch <- []interface{}{opaque, []byte("response1")}
			donech <- true
		})
	server.SetRequestHandlerFor(
		0x80000001,
		func(opaque uint32, req interface{}, respch chan []interface{}) {
			respch <- []interface{}{opaque, []byte("response2")}
			donech <- true
		})

	// make client
	client, err := NewClient(host, clientConfig, nil)
	if err != nil {
		t.Fatal(err)
	}
	client.SetEncoder(EncodingBinary, nil)
	client.Start()

	do := func(opaque uint32, req, resp interface{}) {
		response, err := client.RequestWith(opaque, flags, req)
		if err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(resp, response) {
			t.Fatalf("mismatch in response %T %s", response, response)
		}
		t.Logf("%v done", string(req.([]byte)))
	}
	go func() {
		for i := 0; i < 1000; i++ {
			do(0x80000000, []byte("request1"), []byte("response1"))
		}
	}()
	go func() {
		for i := 0; i < 1000; i++ {
			do(0x80000001, []byte("request2"), []byte("response2"))
		}
	}()

	for i := 2000; i > 0; i-- {
		<-donech
	}

	time.Sleep(100 * time.Millisecond)
	client.Close()
}

func BenchmarkPostTx(b *testing.B) {
	server.SetPostHandler(postHandler)
	// make client
	client, err := NewClient(host, clientConfig, nil)
	if err != nil {
		b.Fatal(err)
	}
	client.SetEncoder(EncodingBinary, nil)
	client.Start()

	for i := 0; i < b.N; i++ {
		client.Post(flags, postPayload)
		b.SetBytes(int64(len(postPayload)))
	}
	client.Close()
}

func postHandler(opaque uint32, request interface{}) {
	if opaque != OpaquePost {
		log.Fatalf("Post opaque is %v", opaque)
	} else if bytes.Compare(postPayload, request.([]byte)) != 0 {
		log.Fatalf("Post payload is %v", request)
	}
}
