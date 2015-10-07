//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "compress/gzip"
import "bytes"

func make_gzip(t *Transport, config map[string]interface{}) (uint64, tagfn, tagfn) {
	buffersize := config["buffersize"].(int)
	level := config["gzip.level"].(int)
	wbuf := bytes.NewBuffer(make([]byte, 0, buffersize))
	writer, err := gzip.NewWriterLevel(wbuf, level)
	if err != nil {
		panic(err)
	}
	enc := func(in, out []byte) int {
		if len(in) == 0 { // empty input
			return 0
		}
		wbuf.Reset()
		writer.Reset(wbuf)
		_, err := writer.Write(in)
		if err != nil {
			panic(err)
		} else if err = writer.Flush(); err != nil {
			panic(err)
		}
		return copy(out, wbuf.Bytes())
	}
	dec := func(in, out []byte) int {
		if len(in) == 0 {
			return 0
		}
		reader, err := gzip.NewReader(bytes.NewBuffer(in))
		if err != nil {
			panic(err)
		}
		n, err := reader.Read(out)
		if err != nil {
			panic(err)
		}
		return n
	}
	return tagGzip, enc, dec
}

func init() {
	tag_factory["gzip"] = make_lzw
}
