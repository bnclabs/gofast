//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "compress/gzip"
import "bytes"
import "fmt"

func make_gzip(t *Transport, config map[string]interface{}) (uint64, tagfn, tagfn) {
	enc := func(in, out []byte) int {
		level := config["gzip.level"].(int)
		if len(in) == 0 { // empty input
			return 0
		}
		wbuf := bytes.NewBuffer(out[:0])
		writer, err := gzip.NewWriterLevel(wbuf, level)
		if err != nil {
			panic(fmt.Errorf("error encoding gzip: %v", err))
		}
		_, err = writer.Write(in)
		if err != nil {
			panic(fmt.Errorf("error encoding gzip: %v", err))
		} else if err = writer.Flush(); err != nil {
			panic(fmt.Errorf("error encoding gzip: %v", err))
		}
		return wbuf.Len()
	}
	dec := func(in, out []byte) int {
		if len(in) == 0 {
			return 0
		}
		reader, err := gzip.NewReader(bytes.NewBuffer(in))
		if err != nil {
			panic(fmt.Errorf("error decoding gzip: %v", err))
		}
		n, err := reader.Read(out)
		if err != nil {
			panic(fmt.Errorf("error decoding gzip: %v", err))
		}
		return n
	}
	return tagGzip, enc, dec
}

func init() {
	tag_factory["gzip"] = make_gzip
}
