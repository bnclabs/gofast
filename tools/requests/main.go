package main

import "sync"
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
	fname := "requests.pprof"
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
	}

	// take memory profile.
	fname = "requests.mprof"
	fd, err = os.Create(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()
	pprof.WriteHeapProfile(fd)
}

func doTransport(callb func(trans *gofast.Transport)) {
	var wg sync.WaitGroup

	n_trans := make([]*gofast.Transport, 0, 16)
	for i := 0; i < options.conns; i++ {
		wg.Add(1)
		ver := testVersion(1)
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
		trans.SubscribeMessage(&msgPost{}, nil)
		trans.Handshake()
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

	for i := 0; i < options.routines; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < options.count; j++ {
				data := make([]byte, options.payload+12)
				for i := 0; i < options.payload; i++ {
					data[i] = 'a'
				}
				pd := options.payload
				since := time.Now()
				tmp := strconv.AppendInt(data[pd:pd], int64(j), 10)
				msg := newMsgPost(data[:pd+len(tmp)])
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
	time.Sleep(1 * time.Second)
}

func doRequest(trans *gofast.Transport) {
	var wg sync.WaitGroup

	echo := make([]byte, options.payload+12)
	for i := 0; i < options.payload; i++ {
		echo[i] = 'a'
	}
	pd := options.payload

	for i := 0; i < options.routines; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < options.count; j++ {
				since := time.Now()
				tmp := strconv.AppendInt(echo[pd:pd], int64(j), 10)
				s := string(echo[:pd+len(tmp)])
				if ping, err := trans.Ping(s); err != nil {
					fmt.Printf("%v\n", err)
					panic("exit")
				} else if got := ping.Repr(); got != s {
					fmt.Printf("expected %v, got %v\n", s, got)
					panic("exit")
				}
				av.Add(uint64(time.Since(since)))
			}
			wg.Done()
		}()
	}
	wg.Wait()
	time.Sleep(1 * time.Second)
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
		msg.data = make([]byte, options.payload)
	}
	copy(msg.data, in[m:m+ln])
}

func (msg *msgPost) String() string {
	return "msgPost"
}
