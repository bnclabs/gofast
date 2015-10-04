//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "compress/lzw"
import "bytes"
import "io"

func make_lzw(t *Transport, config map[string]interface{}) (tagfn, tagfn) {
	if hasString("lzw", t.tags) == false {
		return nil, nil
	}

	enc := func(in, out []byte) int {
		wbuf := bytes.NewBuffer(out[:])
		writer := lzw.NewWriter(wbuf, lzw.LSB, 8 /*litWidth*/)
		_, err := writer.Writer(in)
		if err != nil {
			panic(err)
		}
		writer.Close()
		return copy(out, wbuf.Bytes())
	}
	dec := func(in, out []byte) int {
		reader := lzw.NewReader(bytes.NewReader(in), lzw.LSB, 8 /*litWidth*/)
		n, err := io.ReadFull(reader, out)
		if err != nil {
			panic(err)
		}
		return n
	}
}

func init() {
	tag_factory = make_lzw
}
