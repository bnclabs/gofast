package main

import "sync"
import "sync/atomic"
import "runtime"
import "flag"
import "reflect"
import "unsafe"
import "os"
import "log"
import "sort"
import "time"
import "strings"
import "strconv"
import "net"
import "fmt"
import "compress/flate"
import "runtime/pprof"

import "github.com/prataprc/gofast"

var options struct {
	do       string
	count    int
	routines int
	conns    int
	addr     string
	payload  int
	log      string
}

func argParse() {
	flag.StringVar(&options.do, "do", "post",
		"post / request / stream benchmark to do")
	flag.IntVar(&options.conns, "conns", 1,
		"number of connections to use")
	flag.IntVar(&options.routines, "routines", 1,
		"number of concurrent routines per connection")
	flag.IntVar(&options.count, "count", 1,
		"number of requests per routine")
	flag.StringVar(&options.addr, "addr", "127.0.0.1:9998",
		"number of concurrent routines")
	flag.StringVar(&options.log, "log", "error",
		"number of concurrent routines")
	flag.IntVar(&options.payload, "payload", 10,
		"payload size to ping pong.")
	flag.Parse()
}

var av = &Average{}

func main() {
	argParse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	// start cpu profile.
	fname := "client.pprof"
	fd, err := os.Create(fname)
	if err != nil {
		log.Fatalf("unable to create %q: %v\n", fname, err)
	}
	defer fd.Close()
	pprof.StartCPUProfile(fd)
	defer pprof.StopCPUProfile()

	switch options.do {
	case "post":
		doTransport(doPost)
	case "request":
		doTransport(doRequest)
	case "stream":
		doTransport(doStream)
	}

	// take memory profile.
	fname = "client.mprof"
	fd, err = os.Create(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()
	pprof.WriteHeapProfile(fd)
}

func doTransport(callb func(trans *gofast.Transport)) {
	var wg sync.WaitGroup

	ver := testVersion(1)
	n_trans := make([]*gofast.Transport, 0, 16)
	for i := 0; i < options.conns; i++ {
		wg.Add(1)
		config := newconfig("client", 3000, 4000)
		config["tags"] = ""
		conn, err := net.Dial("tcp", options.addr)
		if err != nil {
			panic(err)
		}
		trans, err := gofast.NewTransport(conn, &ver, nil, config)
		if err != nil {
			panic(err)
		}
		n_trans = append(n_trans, trans)
		go func(trans *gofast.Transport) {
			trans.FlushPeriod(100 * time.Millisecond)
			callb(trans)
			wg.Done()
			trans.Close()
		}(trans)
	}
	wg.Wait()
	printCounts(addCounts(n_trans...))
	fmsg := "request stats: n:%v mean:%v var:%v sd:%v\n"
	n, m := av.Count(), time.Duration(av.Mean())
	v, s := time.Duration(av.Variance()), time.Duration(av.Sd())
	fmt.Printf(fmsg, n, m, v, s)
}

func doPost(trans *gofast.Transport) {
	var wg sync.WaitGroup

	trans.SubscribeMessage(&msgPost{}, nil).Handshake()
	msg := &msgPost{data: make([]byte, options.payload)}
	for i := 0; i < options.payload; i++ {
		msg.data[i] = 'a'
	}

	for i := 0; i < options.routines; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < options.count; j++ {
				since := time.Now()
				if err := trans.Post(msg, false); err != nil {
					fmt.Printf("%v\n", err)
					panic("exit")
				}
				av.Add(uint64(time.Since(since)))
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if _, err := trans.Whoami(); err != nil {
		log.Fatal(err)
	}
}

func doRequest(trans *gofast.Transport) {
	var wg sync.WaitGroup

	trans.SubscribeMessage(&msgReqsp{}, nil).Handshake()
	msg := &msgReqsp{data: make([]byte, options.payload)}
	for i := 0; i < options.payload; i++ {
		msg.data[i] = 'a'
	}

	for i := 0; i < options.routines; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < options.count; j++ {
				since := time.Now()
				if rmsg, err := trans.Request(msg, false); err != nil {
					fmt.Printf("%v\n", err)
					panic("exit")
				} else {
					trans.Free(rmsg)
				}
				av.Add(uint64(time.Since(since)))
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func doStream(trans *gofast.Transport) {
	var wg sync.WaitGroup

	trans.SubscribeMessage(&msgStream{}, nil).Handshake()

	for i := 0; i < options.routines; i++ {
		wg.Add(1)
		go func() {
			msg := &msgStream{data: make([]byte, 0, options.payload*2)}
			ch := make(chan gofast.Message, 100)
			since := time.Now()
			stream, err := trans.Stream(msg, true, ch)
			if err != nil {
				log.Fatal(err)
			}
			//fmt.Println("started", msg)
			var received int64
			go func() {
				for {
					msg, ok := <-ch
					if ok {
						trans.Free(msg)
						atomic.AddInt64(&received, 1)
						if atomic.LoadInt64(&received) == int64(options.count+1) {
							stream.Close()
							break
						}
						//fmt.Println("received", msg)
					} else {
						break
					}
				}
				wg.Done()
			}()
			av.Add(uint64(time.Since(since)))
			for j := 0; j < options.count; j++ {
				msg.data = msg.data[:cap(msg.data)]
				for i := 0; i < options.payload; i++ {
					msg.data[i] = 'a'
				}
				n := options.payload
				tmp := strconv.AppendInt(msg.data[n:n], int64(j), 10)
				msg.data = msg.data[:n+len(tmp)]

				since := time.Now()
				if err := stream.Stream(msg, true); err != nil {
					fmt.Printf("%v\n", err)
					panic("exit")
				}
				//fmt.Println("count", msg)
				av.Add(uint64(time.Since(since)))
			}
		}()
	}
	wg.Wait()
}

func newconfig(name string, start, end int) map[string]interface{} {
	return map[string]interface{}{
		"name":         name,
		"buffersize":   1024,
		"chansize":     100000,
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

func addCounts(n_trans ...*gofast.Transport) map[string]uint64 {
	counts := n_trans[0].Counts()
	for _, trans := range n_trans[1:] {
		for k, v := range trans.Counts() {
			counts[k] += v
		}
	}
	return counts
}

func bytes2str(bytes []byte) string {
	if bytes == nil {
		return ""
	}
	sl := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	st := &reflect.StringHeader{Data: sl.Data, Len: sl.Len}
	return *(*string)(unsafe.Pointer(st))
}

//-- post message for benchmarking

type msgPost struct {
	data []byte
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
		msg.data = make([]byte, 0, options.payload)
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
		msg.data = make([]byte, 0, options.payload)
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
		msg.data = make([]byte, 0, options.payload)
	}
	copy(msg.data[:0], in[m:m+ln])
}

func (msg *msgStream) String() string {
	return "msgStream"
}
