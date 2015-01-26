//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "compress/gzip"
import "bytes"

// GzipCompression is default handler for gzip compression.
type GzipCompression struct {
	reader  *gzip.Reader
	writer  *gzip.Writer
	wbuffer bytes.Buffer
}

// NewGzipCompression returns a new instance of gzip-compression.
func NewGzipCompression() *GzipCompression {
	return &GzipCompression{
		reader: nil,
		writer: nil,
	}
}

// Zip implements Compressor{} interface.
func (gz *GzipCompression) Zip(in, out []byte) (data []byte, err error) {
	if gz.writer == nil {
		w, err := gzip.NewWriterLevel(&gz.wbuffer, gzip.DefaultCompression)
		if err != nil {
			return nil, err
		}
		gz.writer = w
	} else {
		gz.wbuffer.Reset()
		gz.writer.Reset(&gz.wbuffer)
	}
	_, err = gz.writer.Write(in)
	if err != nil {
		return nil, err
	} else if err := gz.writer.Flush(); err != nil {
		return nil, err
	}
	return gz.wbuffer.Bytes(), nil
}

// Unzip implements Compressor{} interface.
func (gz *GzipCompression) Unzip(in, out []byte) (data []byte, err error) {
	if gz.reader == nil {
		r, err := gzip.NewReader(bytes.NewReader(in))
		if err != nil {
			return nil, err
		}
		gz.reader = r
	} else {
		gz.reader.Reset(bytes.NewReader(in))
	}
	n, err := gz.reader.Read(out)
	return out[:n], err
}
