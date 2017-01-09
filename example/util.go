package main

import "reflect"
import "fmt"
import "sort"
import "strings"
import "compress/flate"
import "unsafe"

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

func printCounts(counts map[string]uint64) {
	if counts == nil {
		fmt.Println("statistics is nil")
		return
	}
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
}

func addCounts(n_trans ...*gofast.Transport) map[string]uint64 {
	if len(n_trans) > 0 {
		counts := n_trans[0].Stat()
		for _, trans := range n_trans[1:] {
			for k, v := range trans.Stat() {
				counts[k] += v
			}
		}
		return counts
	}
	return nil
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
