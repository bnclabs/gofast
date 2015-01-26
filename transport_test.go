//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "testing"
import "net"
import "reflect"
import "bytes"
import "io/ioutil"

var binaryPayload, _ = ioutil.ReadFile("./transport.go")

func TestEncodingBinary(t *testing.T) {
	LogIgnore()
	//SetLogLevel(LogLevelTrace)

	flags := TransportFlag(EncodingBinary)
	flags = flags.SetRequest().SetStream().SetEndStream()

	// new transport packet
	tc := newTestConnection().reset()
	pkt := NewTransportPacket(tc, 1000*1024, NewSystemLog())
	pkt.SetEncoder(EncodingBinary, nil)

	mtype, opaque := MtypeBinaryPayload, uint32(0xdeadbeaf)
	payload := []byte("encoding-binary")
	if err := pkt.Send(mtype, flags, opaque, payload); err != nil {
		t.Fatal(err)
	}
	if mtype, rflags, ropaque, rpayload, err := pkt.Receive(); err != nil {
		t.Fatal(err)
	} else if mtype != MtypeBinaryPayload {
		t.Fatalf("Mismatch in mtype %x", mtype)
	} else if flags != rflags {
		t.Fatalf("Mismatch in flags %x", rflags)
	} else if opaque != ropaque {
		t.Fatalf("Mismatch in opaque %x", ropaque)
	} else if bytes.Compare(payload, rpayload.([]byte)) != 0 {
		t.Fatalf("Mismatch in payload %v", rpayload)
	}
}

func TestCompressions(t *testing.T) {
	LogIgnore()
	//SetLogLevel(LogLevelTrace)

	// new transport packet
	tc := newTestConnection().reset()
	pkt := NewTransportPacket(tc, 1000*1024, NewSystemLog())
	mtype, opaque := MtypeBinaryPayload, uint32(0xdeadbeaf)
	payload := binaryPayload
	flags := TransportFlag(0).SetBinary().SetGzip()

	verify := func() {
		if err := pkt.Send(mtype, flags, opaque, payload); err != nil {
			t.Fatal(err)
		}
		if mtype, rflags, ropaque, rpayload, err := pkt.Receive(); err != nil {
			t.Fatal(err)
		} else if mtype != MtypeBinaryPayload {
			t.Fatalf("Mismatch in mtype %x", mtype)
		} else if flags != rflags {
			t.Fatalf("Mismatch in flags %x", rflags)
		} else if opaque != ropaque {
			t.Fatalf("Mismatch in opaque %x", ropaque)
		} else if reflect.DeepEqual(payload, rpayload.([]byte)) == false {
			t.Fatalf("Mismatch in payload %v", 10)
		}
	}

	// Check for gzip using transport instance.
	flags = TransportFlag(0).SetBinary().SetGzip()
	flags = flags.SetRequest().SetStream().SetEndStream()
	pkt.SetEncoder(EncodingBinary, nil)
	pkt.SetZipper(CompressionGzip, nil)
	verify()

	// Check for lzw using same transport instance.
	flags = TransportFlag(0).SetBinary().SetLZW()
	flags = flags.SetRequest().SetStream().SetEndStream()
	pkt.SetEncoder(EncodingBinary, nil)
	pkt.SetZipper(CompressionLZW, nil)
	verify()

	// Check for gzip using same transport instance.
	flags = TransportFlag(0).SetBinary().SetGzip()
	flags = flags.SetRequest().SetStream().SetEndStream()
	pkt.SetEncoder(EncodingBinary, nil)
	pkt.SetZipper(CompressionGzip, nil)
	verify()

	// Check for lzw using same transport instance.
	flags = TransportFlag(0).SetBinary().SetLZW()
	flags = flags.SetRequest().SetStream().SetEndStream()
	pkt.SetEncoder(EncodingBinary, nil)
	pkt.SetZipper(CompressionLZW, nil)
	verify()
}

func BenchmarkBinaryTx(b *testing.B) {
	flags := TransportFlag(EncodingBinary)
	flags = flags.SetRequest().SetStream().SetEndStream()

	// new transport packet
	tc := newTestConnection().reset()
	pkt := NewTransportPacket(tc, 1000*1024, NewSystemLog())
	pkt.SetEncoder(EncodingBinary, nil)

	mtype, opaque := uint16(0x0001), uint32(0xdeadbeaf)
	payload := binaryPayload
	for i := 0; i < b.N; i++ {
		if err := pkt.Send(mtype, flags, opaque, payload); err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(len(binaryPayload)))
		tc = tc.reset()
	}
}

func BenchmarkBinaryRx(b *testing.B) {
	flags := TransportFlag(EncodingBinary)
	flags = flags.SetBinary().SetRequest().SetStream().SetEndStream()

	// new transport packet
	tc := newTestConnection().reset()
	pkt := NewTransportPacket(tc, 1000*1024, NewSystemLog())
	pkt.SetEncoder(EncodingBinary, nil)

	mtype, opaque := uint16(0x0001), uint32(0xdeadbeaf)
	payload := binaryPayload
	for i := 0; i < b.N; i++ {
		if err := pkt.Send(mtype, flags, opaque, payload); err != nil {
			b.Fatal(err)
		}
		if _, _, _, _, err := pkt.Receive(); err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(len(binaryPayload) * 2))
		tc = tc.reset()
	}
}

func BenchmarkGzipTx(b *testing.B) {
	flags := TransportFlag(0).SetBinary().SetGzip()
	flags = flags.SetRequest().SetStream().SetEndStream()

	// new transport packet
	tc := newTestConnection().reset()
	pkt := NewTransportPacket(tc, 1000*1024, NewSystemLog())
	pkt.SetEncoder(EncodingBinary, nil)
	pkt.SetZipper(CompressionGzip, nil)

	mtype, opaque := uint16(0x0001), uint32(0xdeadbeaf)
	payload := binaryPayload
	for i := 0; i < b.N; i++ {
		if err := pkt.Send(mtype, flags, opaque, payload); err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(tc.woff))
		tc = tc.reset()
	}
}

func BenchmarkGzipRx(b *testing.B) {
	flags := TransportFlag(0).SetBinary().SetGzip()
	flags = flags.SetRequest().SetStream().SetEndStream()

	// new transport packet
	tc := newTestConnection().reset()
	pkt := NewTransportPacket(tc, 1000*1024, NewSystemLog())
	pkt.SetEncoder(EncodingBinary, nil)
	pkt.SetZipper(CompressionGzip, nil)

	mtype, opaque := uint16(0x0001), uint32(0xdeadbeaf)
	payload := binaryPayload
	for i := 0; i < b.N; i++ {
		if err := pkt.Send(mtype, flags, opaque, payload); err != nil {
			b.Fatal(err)
		}
		if _, _, _, _, err := pkt.Receive(); err != nil {
			b.Fatal(err)
		}
		tc = tc.reset()
	}
}

func BenchmarkLZWTx(b *testing.B) {
	flags := TransportFlag(0).SetBinary().SetLZW()
	flags = flags.SetRequest().SetStream().SetEndStream()

	// new transport packet
	tc := newTestConnection().reset()
	pkt := NewTransportPacket(tc, 1000*1024, NewSystemLog())
	pkt.SetEncoder(EncodingBinary, nil)
	pkt.SetZipper(CompressionLZW, nil)

	mtype, opaque := uint16(0x0001), uint32(0xdeadbeaf)
	payload := binaryPayload
	for i := 0; i < b.N; i++ {
		if err := pkt.Send(mtype, flags, opaque, payload); err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(tc.woff))
		tc = tc.reset()
	}
}

func BenchmarkLZWRx(b *testing.B) {
	flags := TransportFlag(0).SetBinary().SetLZW()
	flags = flags.SetRequest().SetStream().SetEndStream()

	// new transport packet
	tc := newTestConnection().reset()
	pkt := NewTransportPacket(tc, 1000*1024, NewSystemLog())
	pkt.SetEncoder(EncodingBinary, nil)
	pkt.SetZipper(CompressionLZW, nil)

	mtype, opaque := uint16(0x0001), uint32(0xdeadbeaf)
	payload := binaryPayload
	for i := 0; i < b.N; i++ {
		if err := pkt.Send(mtype, flags, opaque, payload); err != nil {
			b.Fatal(err)
		}
		if _, _, _, _, err := pkt.Receive(); err != nil {
			b.Fatal(err)
		}
		tc = tc.reset()
	}
}

type testConnection struct {
	roff  int
	woff  int
	buf   []byte
	laddr netAddr
	raddr netAddr
}

func newTestConnection() *testConnection {
	return &testConnection{
		buf:   make([]byte, 100000),
		laddr: netAddr("127.0.0.1:9998"),
		raddr: netAddr("127.0.0.1:9999"),
	}
}

func (tc *testConnection) Write(b []byte) (n int, err error) {
	newoff := tc.woff + len(b)
	copy(tc.buf[tc.woff:newoff], b)
	tc.woff = newoff
	return len(b), nil
}

func (tc *testConnection) Read(b []byte) (n int, err error) {
	newoff := tc.roff + len(b)
	copy(b, tc.buf[tc.roff:newoff])
	tc.roff = newoff
	return len(b), nil
}

func (tc *testConnection) LocalAddr() net.Addr {
	return tc.laddr
}

func (tc *testConnection) RemoteAddr() net.Addr {
	return tc.raddr
}

func (tc *testConnection) reset() *testConnection {
	tc.woff, tc.roff = 0, 0
	return tc
}

type netAddr string

func (addr netAddr) Network() string {
	return "tcp"
}

func (addr netAddr) String() string {
	return string(addr)
}
