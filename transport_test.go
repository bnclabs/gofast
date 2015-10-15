package gofast

import "testing"
import "fmt"
import "compress/flate"
import "net"
import "time"
import "sync"

func TestTransport(t *testing.T) {
	// init
	laddr, raddr, ver := "127.0.0.1:9998", "127.0.0.1:9999", testVersion(1)
	config := newconfig("testtransport", tagOpaqueStart, tagOpaqueStart+10)
	config["tags"] = "gzip"
	tconn := newTestConnection(laddr, raddr, nil, true)
	trans, err := NewTransport(tconn, &ver, nil, config)
	if err != nil {
		t.Error(err)
	}
	trans.Handshake()
	// test
	if _, ok := trans.tagenc[tagGzip]; !ok && len(trans.tagenc) != 1 {
		t.Errorf("expected gzip, got %v", trans.tagenc)
	}
	if ref := "testtransport"; ref != trans.Name() {
		t.Errorf("expected %v, got %v", ref, trans.Name())
	} else if !trans.PeerVersion().Equal(&ver) {
		t.Errorf("expected %v, got %v", ver, trans.peerver)
	} else if s := trans.LocalAddr().String(); s != laddr {
		t.Errorf("expected %v, got %v", s, laddr)
	} else if s := trans.RemoteAddr().String(); s != raddr {
		t.Errorf("expected %v, got %v", s, raddr)
	}
	trans.Close()
}

func TestSubscribeMessage(t *testing.T) {
	// init
	laddr, raddr, ver := "127.0.0.1:9998", "127.0.0.1:9999", testVersion(1)
	config := newconfig("testtransport", tagOpaqueStart, tagOpaqueStart+10)
	tconn := newTestConnection(laddr, raddr, nil, true)
	trans, err := NewTransport(tconn, &ver, nil, config)
	if err != nil {
		t.Error(err)
	}
	trans.Handshake()
	// test
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic")
			}
		}()
		trans.SubscribeMessage(NewPing("should faile"), nil)
	}()
}

func TestCount(t *testing.T) {
	// init
	laddr, raddr, ver := "127.0.0.1:9998", "127.0.0.1:9999", testVersion(1)
	config := newconfig("testtransport", tagOpaqueStart, tagOpaqueStart+10)
	tconn := newTestConnection(laddr, raddr, nil, true)
	trans, err := NewTransport(tconn, &ver, nil, config)
	if err != nil {
		t.Error(err)
	}
	trans.Handshake()
	// test
	if count, ref := len(trans.Counts()), 19; count != ref {
		t.Errorf("expected %v, got %v", ref, count)
	}
}

//func TestFlushPeriod(t *testing.T) {
//	// init
//	laddr, raddr, ver := "127.0.0.1:9998", "127.0.0.1:9999", testVersion(1)
//	config := newconfig("testtransport", tagOpaqueStart, tagOpaqueStart+10)
//	tconn := newTestConnection(laddr, raddr, nil, true)
//	trans, err := NewTransport(tconn, &ver, nil, config)
//	if err != nil {
//		t.Error(err)
//	}
//	trans.Handshake()
//	// test
//	trans.FlushPeriod(10 * time.Millisecond)
//	time.Sleep(1 * time.Second)
//	if trans.n_flushes != 100
//}

func BenchmarkTransCounts(b *testing.B) {
	// init
	laddr, raddr, ver := "127.0.0.1:9998", "127.0.0.1:9999", testVersion(1)
	config := newconfig("testtransport", tagOpaqueStart, tagOpaqueStart+10)
	tconn := newTestConnection(laddr, raddr, nil, true)
	trans, err := NewTransport(tconn, &ver, nil, config)
	if err != nil {
		b.Error(err)
	}
	trans.Handshake()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trans.Counts()
	}
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
		"log.level":    "error",
		"gzip.file":    flate.BestSpeed,
	}
}

type testConnection struct {
	roff  int
	woff  int
	buf   []byte
	read  bool
	mu    sync.Mutex
	laddr netAddr
	raddr netAddr
}

func newTestConnection(l, r string, buf []byte, read bool) *testConnection {
	tconn := &testConnection{
		laddr: netAddr(l),
		raddr: netAddr(r),
		read:  read,
	}
	if tconn.buf = buf; buf == nil {
		tconn.buf = make([]byte, 100000)
	}
	return tconn
}

func (tc *testConnection) Write(b []byte) (n int, err error) {
	do := func() (int, error) {
		tc.mu.Lock()
		defer tc.mu.Unlock()
		newoff := tc.woff + len(b)
		copy(tc.buf[tc.woff:newoff], b)
		tc.woff = newoff
		return len(b), nil
	}
	n, err = do()
	//fmt.Println("write ...", n, err)
	return
}

func (tc *testConnection) Read(b []byte) (n int, err error) {
	do := func() (int, error) {
		tc.mu.Lock()
		defer tc.mu.Unlock()
		if newoff := tc.roff + len(b); newoff <= tc.woff {
			copy(b, tc.buf[tc.roff:newoff])
			tc.roff = newoff
			return len(b), nil
		}
		return 0, nil
	}
	for err == nil && n == 0 {
		if tc.read {
			n, err = do()
		}
		time.Sleep(100 * time.Millisecond)
	}
	//fmt.Println("read ...", n, err)
	return n, err
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

func (tx *testConnection) Close() error {
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

func (v *testVersion) Less(ver Version) bool {
	return (*v) < (*ver.(*testVersion))
}

func (v *testVersion) Equal(ver Version) bool {
	return (*v) == (*ver.(*testVersion))
}

func (v *testVersion) String() string {
	return fmt.Sprintf("%v", int(*v))
}

func (v *testVersion) Marshal(out []byte) int {
	return valuint642cbor(uint64(*v), out)
}

func (v *testVersion) Unmarshal(in []byte) int {
	ln, n := cborItemLength(in)
	*v = testVersion(ln)
	return n
}
