package gofast

import "testing"
import "fmt"
import "bytes"
import "compress/flate"
import "net"

func TestTransport(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+1000
	config, conn := newconfig("testtransport", st, end), newTestConnection()
	trans, err := NewTransport(conn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(trans)
}

func newconfig(name string, start, end int) map[string]interface{} {
	return map[string]interface{}{
		"name":         name,
		"buffersize":   1024 * 1024 * 10,
		"chansize":     1,
		"batchsize":    1,
		"tags":         "",
		"opaque.start": start,
		"opaque.end":   end,
		"log.level":    "debug",
		"gzip.file":    flate.BestSpeed,
	}
}

type testConnection struct {
	buf   bytes.Buffer
	laddr netAddr
	raddr netAddr
}

func newTestConnection() *testConnection {
	return &testConnection{
		laddr: netAddr("127.0.0.1:9998"),
		raddr: netAddr("127.0.0.1:9999"),
	}
}

func (tc *testConnection) Write(b []byte) (n int, err error) {
	return tc.buf.Write(b)
}

func (tc *testConnection) Read(b []byte) (n int, err error) {
	return tc.buf.Read(b)
}

func (tc *testConnection) LocalAddr() net.Addr {
	return tc.laddr
}

func (tc *testConnection) RemoteAddr() net.Addr {
	return tc.raddr
}

func (tc *testConnection) Close() error {
	tc.buf.Reset()
	return nil
}

type netAddr string

func (addr netAddr) Network() string {
	return "tcp"
}

func (addr netAddr) String() string {
	return string(addr)
}

type testVersion int

func (v testVersion) Less(ver Version) bool {
	return v < ver.(testVersion)
}

func (v testVersion) Equal(ver Version) bool {
	return v == ver.(testVersion)
}

func (v testVersion) String() string {
	return fmt.Sprintf("%v", v)
}

func (v testVersion) Value() interface{} {
	return int(v)
}
