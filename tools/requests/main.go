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
	count    int
	routines int
	conns    int
	addr     string
	log      string
}

func argParse() {
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

	var wg sync.WaitGroup
	n_trans := make([]*gofast.Transport, 0)
	for i := 0; i < options.conns; i++ {
		wg.Add(1)
		go func() {
			config := newconfig("client", 267, 277)
			config["tags"] = ""
			conn, err := net.Dial("tcp", options.addr)
			if err != nil {
				panic(err)
			}
			ver := testVersion(1)

			trans, err := gofast.NewTransport(conn, &ver, nil, config)
			if err != nil {
				panic(err)
			}
			n_trans = append(n_trans, trans)
			doRequest(trans)
			wg.Done()
			trans.Close()
		}()
	}
	wg.Wait()
	printCounts(addCounts(n_trans...))
	fmsg := "request stats: n:%v mean:%v var:%v sd:%v\n"
	n, m := av.Count(), time.Duration(av.Mean())
	v, s := time.Duration(av.Variance()), time.Duration(av.Sd())
	fmt.Printf(fmsg, n, m, v, s)

	// take memory profile.
	fname = "requests.mprof"
	fd, err = os.Create(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()
	pprof.WriteHeapProfile(fd)
}

func doRequest(trans *gofast.Transport) {
	var wg sync.WaitGroup

	for i := 0; i < options.routines; i++ {
		wg.Add(1)
		go func() {
			var echo [64]byte
			n := copy(echo[:], "ping")
			for j := 0; j < options.count; j++ {
				since := time.Now()
				s := bytes2str(strconv.AppendInt(echo[n:], int64(j), 10))
				if ping, err := trans.Ping(s); err != nil {
					log.Fatal(err)
				} else if got := ping.Repr(); got != s {
					log.Fatalf("expected %v, got %v\n", s, got)
				}
				av.Add(uint64(time.Since(since)))
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func newconfig(name string, start, end int) map[string]interface{} {
	return map[string]interface{}{
		"name":         name,
		"buffersize":   1024 * 1,
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
