package main

import "time"
import "runtime"
import "flag"
import "sort"
import "strings"
import "net"
import "fmt"
import "compress/flate"

import "github.com/prataprc/gofast"

var options struct {
	addr string
	log  string
}

func argParse() {
	flag.StringVar(&options.addr, "addr", "127.0.0.1:9998",
		"number of concurrent routines")
	flag.StringVar(&options.log, "log", "error",
		"number of concurrent routines")
	flag.Parse()
}

func main() {
	argParse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	config := newconfig("server", 256, 266)
	config["tags"] = ""

	lis, err := net.Listen("tcp", options.addr)
	if err != nil {
		panic(fmt.Errorf("listen failed %v", err))
	}
	fmt.Printf("listening on %v\n", options.addr)
	for {
		if conn, err := lis.Accept(); err == nil {
			ver := testVersion(1)
			trans, err := gofast.NewTransport(conn, &ver, nil, config)
			if err != nil {
				panic("NewTransport server failed")
			}
			go func(trans *gofast.Transport) {
				fmt.Println("new transport", conn.RemoteAddr(), conn.LocalAddr())
				tick := time.Tick(1 * time.Second)
				for {
					<-tick
					printCounts(trans.Counts())
					if _, err := trans.Ping("ok"); err != nil {
						trans.Close()
						return
					}
				}
			}(trans)
		}
	}
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
