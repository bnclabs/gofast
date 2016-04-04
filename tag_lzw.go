//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "compress/lzw"
import "bytes"
import "io"
import "fmt"

func make_lzw(t *Transport, config map[string]interface{}) (uint64, tagfn, tagfn) {
	var wbuf bytes.Buffer
	enc := func(in, out []byte) int {
		if len(in) == 0 {
			return 0
		}
		wbuf.Reset()
		writer := lzw.NewWriter(&wbuf, lzw.LSB, 8 /*litWidth*/)
		if _, err := writer.Write(in); err != nil {
			panic(fmt.Errorf("error encoding lzw: %v", err))
		}
		writer.Close()
		return copy(out, wbuf.Bytes())
	}
	dec := func(in, out []byte) int {
		if len(in) == 0 {
			return 0
		}
		reader := lzw.NewReader(bytes.NewReader(in), lzw.LSB, 8 /*litWidth*/)
		n, err := readAll(reader, out)
		if err != nil {
			panic(fmt.Errorf("error decoding lzw: %v", err))
		}
		reader.Close()
		return n
	}
	return tagLzw, enc, dec
}

func readAll(r io.Reader, out []byte) (n int, err error) {
	c := 0
	for err == nil {
		// Per http://golang.org/pkg/io/#Reader, it is valid for Read to
		// return EOF with non-zero number of bytes at the end of the
		// input stream
		c, err = r.Read(out[n:])
		n += c
	}
	if err == io.EOF {
		return n, nil
	}
	return n, err
}

func init() {
	tag_factory["lzw"] = make_lzw
}
