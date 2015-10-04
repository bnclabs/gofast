//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "compress/gzip"
import "compress/flate"
import "bytes"

func make_gzip(t *Transport, config map[string]interface{}) (tagfn, tagfn) {
	if hasString("gzip", t.tags) == false {
		return nil, nil
	}

	buflen := config["buflen"].(int)
	level := config["gzip.level"].(flate.Compression)
	wbuf := bytes.NewBuffer(make([]byte, 0, buflen))
	writer := gzip.NewWriterLevel(wbuf, level)
	reader, err := gzip.NewReader(bytes.NewReader([]byte{}))
	if err != nil {
		panic(err)
	}
	enc := func(in, out []byte) int {
		wbuf.Reset()
		writer.Reset(&wbuf)
		_, err := writer.Write(in)
		if err != nil {
			panic(err)
		} else if err = writer.Flush(); err != nil {
			panic(err)
		}
		return copy(out, wbuf.Bytes())
	}
	dec := func(in, out []byte) int {
		reader.Reset(bytes.NewReader(in))
		n, err := reader.Read(out)
		if err != nil {
			panic(err)
		}
		return n
	}
	return enc, dec
}

func init() {
	tag_factory = make_lzw
}
