package main

import "time"
import "runtime"
import "flag"
import "io"
import "os"
import "log"
import "bufio"
import "sort"
import "strings"
import "net"
import "fmt"
import "compress/flate"
import "runtime/pprof"

import "github.com/prataprc/gofast"

var options struct {
	cpu  int
	addr string
	log  string
}

func argParse() {
	flag.IntVar(&options.cpu, "cpu", runtime.NumCPU(),
		"GOMAXPROCS")
	flag.StringVar(&options.addr, "addr", "127.0.0.1:9998",
		"number of concurrent routines")
	flag.StringVar(&options.log, "log", "error",
		"number of concurrent routines")
	flag.Parse()
}

func main() {
	argParse()
	runtime.GOMAXPROCS(options.cpu)

	// start cpu profile.
	fname := "server.pprof"
	fd, err := os.Create(fname)
	if err != nil {
		log.Fatalf("unable to create %q: %v\n", fname, err)
	}
	defer fd.Close()
	pprof.StartCPUProfile(fd)
	defer pprof.StopCPUProfile()

	lis, err := net.Listen("tcp", options.addr)
	if err != nil {
		panic(fmt.Errorf("listen failed %v", err))
	}
	fmt.Printf("listening on %v\n", options.addr)

	go runserver(lis)

	fmt.Printf("Press CTRL-D to exit\n")
	reader := bufio.NewReader(os.Stdin)
	_, err = reader.ReadString('\n')
	for err != io.EOF {
		_, err = reader.ReadString('\n')
	}

	// take memory profile.
	fname = "server.mprof"
	fd, err = os.Create(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()
	pprof.WriteHeapProfile(fd)
}

func runserver(lis net.Listener) {
	ver := testVersion(1)
	config := newconfig("server", 1000, 2000)
	config["tags"] = ""
	for {
		if conn, err := lis.Accept(); err == nil {
			fmt.Println("new transport", conn.RemoteAddr(), conn.LocalAddr())
			trans, err := gofast.NewTransport(conn, &ver, nil, config)
			if err != nil {
				panic("NewTransport server failed")
			}
			go func(trans *gofast.Transport) {
				trans.FlushPeriod(100 * time.Millisecond)
				trans.SubscribeMessage(
					&msgPost{},
					func(s *gofast.Stream,
						msg gofast.Message) chan gofast.Message {

						trans.Free(msg)
						return nil
					})
				trans.SubscribeMessage(
					&msgReqsp{},
					func(s *gofast.Stream,
						msg gofast.Message) chan gofast.Message {

						if err := s.Response(msg, false); err != nil {
							log.Fatal(err)
						}
						trans.Free(msg)
						return nil
					})
				trans.SubscribeMessage(
					&msgStream{},
					func(s *gofast.Stream,
						msg gofast.Message) chan gofast.Message {

						ch := make(chan gofast.Message, 100)
						if err := s.Stream(msg, false); err != nil {
							log.Fatal(err)
						}
						trans.Free(msg)
						//fmt.Println("started", msg)
						go func() {
							for msg := range ch {
								//fmt.Println("count", msg)
								if err := s.Stream(msg, false); err != nil {
									log.Fatal(err)
								}
								trans.Free(msg)
							}
							s.Close()
						}()
						return ch
					})
				trans.Handshake()
				tick := time.Tick(1 * time.Second)
				for {
					<-tick
					if options.log == "debug" {
						printCounts(trans.Counts())
					}
				}
			}(trans)
		}
	}
}

func newconfig(name string, start, end int) map[string]interface{} {
	return map[string]interface{}{
		"name":         name,
		"buffersize":   1024,
		"chansize":     1000,
		"batchsize":    100,
		"tags":         "",
		"opaque.start": start,
		"opaque.end":   end,
		"log.level":    options.log,
		"gzip.file":    flate.BestSpeed,
	}
}

type testVersion int

func (v *testVersion) Less(ver gofast.Version) bool {
	return (*v) < (*ver.(*testVersion))
}

func (v *testVersion) Equal(ver gofast.Version) bool {
	return (*v) == (*ver.(*testVersion))
}

func (v *testVersion) String() string {
	return fmt.Sprintf("%v", int(*v))
}

func (v *testVersion) Marshal(out []byte) int {
	return valuint642cbor(uint64(*v), out)
}

func (v *testVersion) Unmarshal(in []byte) int {
	ln, n := cborItemLength(in)
	*v = testVersion(ln)
	return n
}

func printCounts(counts map[string]uint64) {
	keys := []string{}
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Sort(sort.StringSlice(keys))
	s := []string{}
	for _, key := range keys {
		s = append(s, fmt.Sprintf("%v:%v", key, counts[key]))
	}
	fmt.Println(strings.Join(s, ", "))
}

//-- post message for benchmarking

type msgPost struct {
	data []byte
}

func newMsgPost(data []byte) *msgPost {
	return &msgPost{data: data}
}

func (msg *msgPost) Id() uint64 {
	return 111
}

func (msg *msgPost) Encode(out []byte) int {
	return valbytes2cbor(msg.data, out)
}

func (msg *msgPost) Decode(in []byte) {
	ln, m := cborItemLength(in)
	if msg.data == nil {
		msg.data = make([]byte, 0, 64)
	}
	msg.data = append(msg.data[:0], in[m:m+ln]...)
}

func (msg *msgPost) String() string {
	return "msgPost"
}

//-- reqsp message for benchmarking

type msgReqsp struct {
	data []byte
}

func (msg *msgReqsp) Id() uint64 {
	return 112
}

func (msg *msgReqsp) Encode(out []byte) int {
	return valbytes2cbor(msg.data, out)
}

func (msg *msgReqsp) Decode(in []byte) {
	ln, m := cborItemLength(in)
	if msg.data == nil {
		msg.data = make([]byte, 0, 64)
	}
	msg.data = append(msg.data[:0], in[m:m+ln]...)
}

func (msg *msgReqsp) String() string {
	return "msgReqsp"
}

//-- stream message for benchmarking

type msgStream struct {
	data []byte
}

func (msg *msgStream) Id() uint64 {
	return 113
}

func (msg *msgStream) Encode(out []byte) int {
	return valbytes2cbor(msg.data, out)
}

func (msg *msgStream) Decode(in []byte) {
	ln, m := cborItemLength(in)
	if msg.data == nil {
		msg.data = make([]byte, 0, 64)
	}
	msg.data = append(msg.data[:0], in[m:m+ln]...)
}

func (msg *msgStream) String() string {
	return "msgStream"
}
