// +build ignore

package main

import "fmt"
import "log"
import "time"
import "flag"
import "compress/flate"
import "net"

import "github.com/prataprc/gofast"

var options struct {
	laddr   string
	post    int
	request int
	log     string
}

func argParse() {
	flag.StringVar(&options.laddr, "laddr", "localhost:9999",
		"local address to listen")
	flag.IntVar(&options.post, "post", 0,
		"number of post requests to make")
	flag.IntVar(&options.request, "request", 0,
		"number of request-response to make")
	flag.StringVar(&options.log, "log", "warn",
		"set log level, default is warn")
	flag.Parse()
}

func main() {
	argParse()
	config := newconfig("server", 256, 266)
	lis := server(options.laddr, config)
	conn, err := net.Dial("tcp", options.laddr)
	if err != nil {
		log.Fatalf("net dial failed: %v\n", err)
	}
	config = newconfig("client", 256, 266)
	ver := testVersion(1)
	trans, err := gofast.NewTransport(conn, &ver, nil, config)
	if err != nil {
		log.Fatalf("client side NewTransport failed: %v\n", err)
	}
	// handshake with remote
	transinfo(trans)
	trans.Handshake()
	transinfo(trans)
	// start validation
	if options.post > 0 {
		validatePost(trans)
	}
	time.Sleep(2 * time.Second)
	lis.Close()
	trans.Close()
}

func validatePost(trans *gofast.Transport) {
	time.Now()
	for i := 0; i < options.post; i++ {
		hb := gofast.NewHeartbeat(i)
		trans.Post(hb)
	}
	return
}

func server(laddr string, config map[string]interface{}) net.Listener {
	lis, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatalf("listen failed: %v\n", err)
	}

	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				log.Fatalf("accept error: %v\n", err)
			}
			trans, err := gofast.NewTransport(conn, testVersion(1), nil, config)
			if err != nil {
				log.Fatalf("NewTransport failed: %v\n", err)
			}
			go func(trans *gofast.Transport) {
				trans.VersionHandler(testVerhandler)
				tick := time.Tick(1 * time.Second)
				for {
					<-tick
					transinfo(trans)
				}
			}(trans)
		}
	}()
	return lis
}

type testVersion int

func (v testVersion) Less(ver gofast.Version) bool {
	return v < ver.(testVersion)
}

func (v testVersion) Equal(ver gofast.Version) bool {
	return v == ver.(testVersion)
}

func (v testVersion) String() string {
	return fmt.Sprintf("%v", int(v))
}

func (v testVersion) Value() interface{} {
	return int(v)
}

func (v *testVersion) Marshal(out []byte) int {
	return valuint642cbor(uint64(*v), out)
}

func (v *testVersion) Unmarshal(in []byte) int {
	ln, n := cborItemLength(in)
	*v = testVersion(ln)
	return n
}

func testVerhandler(val interface{}) gofast.Version {
	if ver, ok := val.(uint64); ok {
		return testVersion(ver)
	} else if ver, ok := val.(int); ok {
		return testVersion(ver)
	}
	return nil
}

func newconfig(name string, start, end int) map[string]interface{} {
	return map[string]interface{}{
		"name":         name,
		"buffersize":   1024 * 1024 * 10,
		"chansize":     1,
		"batchsize":    1,
		"tags":         "",
		"opaque.start": start,
		"opaque.end":   end,
		"log.level":    options.log,
		"gzip.file":    flate.BestSpeed,
	}
}

func transinfo(trans *gofast.Transport) {
	a, b := trans.LocalAddr(), trans.RemoteAddr()
	c, d := trans.PeerVersion(), trans.Silentsince()
	fmt.Printf("%v -- l:%v ; r:%v ; p:%v ; a:%v\n", trans.Name(), a, b, c, d)
}

func bytes2str(bytes []byte) string {
	if bytes == nil {
		return ""
	}
	sl := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	st := &reflect.StringHeader{Data: sl.Data, Len: sl.Len}
	return *(*string)(unsafe.Pointer(st))
}

func str2bytes(str string) []byte {
	if str == "" {
		return nil
	}
	st := (*reflect.StringHeader)(unsafe.Pointer(&str))
	sl := &reflect.SliceHeader{Data: st.Data, Len: st.Len, Cap: st.Len}
	return *(*[]byte)(unsafe.Pointer(sl))
}
