package gofast

import "testing"
import "time"
import "bytes"
import "fmt"
import "log"

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
	server = makeServer(host)
}

func TestPost(t *testing.T) {
	LogIgnore()
	//SetLogLevel(LogLevelTrace)

	server.SetPostHandler(postHandler)

	// make client
	client, err := NewClient(host, clientConfig, nil)
	if err != nil {
		t.Fatal(err)
	}
	client.SetEncoder(EncodingBinary, nil)
	client.Start()

	client.Post(flags, postPayload)

	client.Close()
	time.Sleep(100 * time.Millisecond)
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

func makeServer(host string) *Server {
	server, err := NewServer(host, serverConfig, nil)
	if err != nil {
		log.Fatal(err)
	}
	server.SetEncoder(EncodingBinary, nil)
	return server
}

func postHandler(opaque uint32, request interface{}) {
	if opaque != OpaquePost {
		log.Fatalf("Post opaque is %v", opaque)
	} else if bytes.Compare(postPayload, request.([]byte)) != 0 {
		log.Fatalf("Post payload is %v", request)
	}
}
