package gofast

import "testing"
import "bytes"
import "time"
import "reflect"

func TestWaiEncode(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		t.Fatal(err)
	}
	transv := <-serverch

	out := make([]byte, 1024)
	ref := []byte{
		6, 99, 108, 105, 101, 110, 116, 1, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0,
	}
	wai := newWhoami(transc)
	if out := wai.Encode(out); bytes.Compare(ref, out) != 0 {
		t.Errorf("expected %v, got %v", ref, out)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestWaiDecode(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		t.Fatal(err)
	}
	transv := <-serverch

	ver, out := testVersion(1), make([]byte, 1024)
	wai, ref := &whoamiMsg{}, newWhoami(transc)
	out = ref.Encode(out)
	wai.version, wai.transport = &ver, transc
	wai.Decode(out)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		t.Fatal(err)
	}
	transv := <-serverch

	wai := newWhoami(transc)
	if wai.String() != "whoamiMsg" {
		t.Errorf("expected whoamiMsg, got %v", wai.String())
	}
	if ref := "client,512"; ref != wai.Repr() {
		t.Errorf("expected %v, got %v", ref, wai.Repr())
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkWaiEncode(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		b.Fatal(err)
	}
	transv := <-serverch

	out := make([]byte, 1024)
	wai := newWhoami(transc)
	for i := 0; i < b.N; i++ {
		out = wai.Encode(out)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkWaiDecode(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		b.Fatal(err)
	}
	transv := <-serverch

	ver, out := testVersion(1), make([]byte, 1024)
	ref := newWhoami(transc)
	ref.tags = "gzip"
	out = ref.Encode(out)
	wai := &whoamiMsg{}
	wai.version, wai.transport = &ver, transc
	for i := 0; i < b.N; i++ {
		wai.Decode(out)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}
