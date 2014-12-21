package gofast

import "compress/lzw"
import "bytes"
import "io"

// LZWCompression is default handler for LZW compression.
type LZWCompression struct{}

// NewLZWCompression returns a new instance of lzw-compression.
func NewLZWCompression() *LZWCompression {
	return &LZWCompression{}
}

// Zip implements Compressor{} interface.
func (z *LZWCompression) Zip(in, out []byte) (data []byte, err error) {
	var wbuffer bytes.Buffer
	writer := lzw.NewWriter(&wbuffer, lzw.LSB, 8 /*litWidth*/)
	_, err = writer.Write(in)
	if err != nil {
		return nil, err
	}
	writer.Close()
	bs := wbuffer.Bytes()
	return bs, nil
}

// Unzip implements Compressor{} interface.
func (z *LZWCompression) Unzip(in, out []byte) (data []byte, err error) {
	reader := lzw.NewReader(bytes.NewReader(in), lzw.LSB, 8 /*litWidth*/)
	n, err := readAll(reader, out)
	reader.Close()
	return out[:n], err
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
