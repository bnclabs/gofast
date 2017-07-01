package main

import "reflect"
import "fmt"
import "os"
import "log"
import "time"
import "sort"
import "strings"
import "unsafe"
import "runtime/pprof"
import "compress/flate"

import "github.com/prataprc/gofast"

func bytes2str(bytes []byte) string {
	if bytes == nil {
		return ""
	}
	sl := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	st := &reflect.StringHeader{Data: sl.Data, Len: sl.Len}
	return *(*string)(unsafe.Pointer(st))
}

func fixbuffer(buffer []byte, size int64) []byte {
	if size == 0 {
		return buffer
	} else if buffer == nil || int64(cap(buffer)) < size {
		return make([]byte, size)
	}
	return buffer[:size]
}

func printCounts(
	counts map[string]uint64, elapsed time.Duration, av *averageInt64) {

	if counts == nil {
		fmt.Println("statistics is nil")
		return
	}
	if options.client {
		// quick bites.
		fmt.Println()
		seconds := elapsed / time.Second
		if seconds < 1 {
			log.Fatalf("Run client for longer duration !!")
		}
		fmt.Printf("Latency Average: %v\n", time.Duration(av.mean()))
		switch options.do {
		case "post":
			throughput := counts["n_txpost"] / uint64(seconds)
			fmt.Printf("Throughput: %v /second\n", throughput)
		case "request":
			throughput := counts["n_txreq"] / uint64(seconds)
			fmt.Printf("Throughput: %v\n", throughput)
		case "streamtx":
			throughput := (counts["n_txstream"] + counts["n_txstart"]) /
				uint64(seconds)
			fmt.Printf("Throughput: %v\n", int64(throughput))
		case "streamrx":
			throughput := (counts["n_rxstream"] + counts["n_rxstart"]) /
				uint64(seconds)
			fmt.Printf("Throughput: %v\n", int64(throughput))
		}
		fmt.Println()
	}

	// sort stats keys and print them.
	keys := []string{}
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Sort(sort.StringSlice(keys))
	s := []string{}
	for _, key := range keys {
		s = append(s, fmt.Sprintf(`"%v":%v`, key, counts[key]))
	}
	fmt.Println("stats {", strings.Join(s, ", "), "}")

	if options.client {
		fmsg := "request stats: n:%v mean:%v var:%v sd:%v\n"
		n, m := av.samples(), time.Duration(av.mean())
		v, s := time.Duration(av.variance()), time.Duration(av.sd())
		fmt.Printf(fmsg, n, m, v, s)
	}
}

func addCounts(transs ...*gofast.Transport) map[string]uint64 {
	if len(transs) > 0 {
		counts := transs[0].Stat()
		for _, trans := range transs[1:] {
			for k, v := range trans.Stat() {
				counts[k] += v
			}
		}
		return counts
	}
	return nil
}

func startCpuProfile(filename string) *os.File {
	fd, err := os.Create(filename)
	if err != nil {
		log.Fatalf("unable to create %q: %v\n", filename, err)
	}
	pprof.StartCPUProfile(fd)
	return fd
}

func doMemProfile(filename string) {
	fd, err := os.Create(filename)
	if err != nil {
		log.Fatalf("unable to create %q: %v\n", filename, err)
	}
	pprof.WriteHeapProfile(fd)
	defer fd.Close()
}

func newconfig(start, end int) map[string]interface{} {
	return map[string]interface{}{
		"buffersize":   options.buffersize,
		"chansize":     100000,
		"batchsize":    100,
		"tags":         "",
		"opaque.start": start,
		"opaque.end":   end,
		"log.level":    options.log,
		"gzip.level":   flate.BestSpeed,
	}
}
