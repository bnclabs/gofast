package gofast

import "testing"
import "bytes"
import "reflect"
import "fmt"

var _ = fmt.Sprintf("dummy")

func TestWaiEncode(t *testing.T) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	out := make([]byte, 1024)
	ref := []byte{
		159, 70, 99, 108, 105, 101, 110, 116, 1, 26, 0, 160, 0, 0, 64, 255,
	}
	wai := NewWhoami(transc)
	if n := wai.Encode(out); bytes.Compare(ref, out[:n]) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestWaiDecode(t *testing.T) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	ver, out := testVersion(1), make([]byte, 1024)
	wai, ref := &Whoami{}, NewWhoami(transc)
	n := ref.Encode(out)
	wai.version, wai.transport = &ver, transc
	wai.Decode(out[:n])
	if !reflect.DeepEqual(ref, wai) {
		t.Errorf("expected %#v, got %#v", ref, wai)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestWhoamiMisc(t *testing.T) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	wai := NewWhoami(transc)
	if wai.String() != "Whoami" {
		t.Errorf("expected Whoami, got %v", wai.String())
	}
	if ref := "client, 10485760"; ref != wai.Repr() {
		t.Errorf("expected %v, got %v", ref, wai.Repr())
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkWaiEncode(b *testing.B) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	out := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		wai := NewWhoami(transc)
		wai.Encode(out)
		whoamipool.Put(wai)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkWaiDecode(b *testing.B) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	ver, out := testVersion(1), make([]byte, 1024)
	ref := NewWhoami(transc)
	ref.tags = []byte("gzip")
	n := ref.Encode(out)
	wai := &Whoami{}
	wai.version, wai.transport = &ver, transc
	for i := 0; i < b.N; i++ {
		wai.Decode(out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}
