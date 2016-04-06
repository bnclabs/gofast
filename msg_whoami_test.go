package gofast

import "testing"
import "bytes"
import "time"
import "reflect"
import "fmt"

var _ = fmt.Sprintf("dummy")

func TestWaiEncode(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	out := make([]byte, 1024)
	ref := []byte{
		159, 70, 99, 108, 105, 101, 110, 116, 1, 25, 2, 0, 64, 255,
	}
	wai := newWhoami(transc)
	if n := wai.Encode(out); bytes.Compare(ref, out[:n]) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestWaiDecode(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	ver, out := testVersion(1), make([]byte, 1024)
	wai, ref := &whoamiMsg{}, newWhoami(transc)
	n := ref.Encode(out)
	wai.version, wai.transport = &ver, transc
	wai.Decode(out[:n])
	if !reflect.DeepEqual(ref, wai) {
		t.Errorf("expected %#v, got %#v", ref, wai)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestWhoamiMisc(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	wai := newWhoami(transc)
	if wai.String() != "whoamiMsg" {
		t.Errorf("expected whoamiMsg, got %v", wai.String())
	}
	if ref := "client, 512"; ref != wai.Repr() {
		t.Errorf("expected %v, got %v", ref, wai.Repr())
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkWaiEncode(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	out := make([]byte, 1024)
	wai := newWhoami(transc)
	for i := 0; i < b.N; i++ {
		wai.Encode(out)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkWaiDecode(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	ver, out := testVersion(1), make([]byte, 1024)
	ref := newWhoami(transc)
	ref.tags = "gzip"
	n := ref.Encode(out)
	wai := &whoamiMsg{}
	wai.version, wai.transport = &ver, transc
	for i := 0; i < b.N; i++ {
		wai.Decode(out[:n])
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}
