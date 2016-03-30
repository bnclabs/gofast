package gofast

import "testing"
import "bytes"
import "reflect"
import "fmt"

var _ = fmt.Sprintf("dummy")

func TestPingEncode(t *testing.T) {
	out := make([]byte, 1024)
	ref := []byte{
		159, 75, 104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 255,
	}
	ping := newPing("hello world")
	if n := ping.Encode(out); bytes.Compare(ref, out[:n]) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func TestPingDecode(t *testing.T) {
	out := make([]byte, 1024)
	ref := newPing("made in india")
	n := ref.Encode(out)
	ping := &pingMsg{}
	ping.Decode(out[:n])
	if !reflect.DeepEqual(ref, ping) {
		t.Errorf("expected %v, got %v", ref, ping)
	}
}

func TestPingMisc(t *testing.T) {
	ping := newPing("hello world")
	if ping.String() != "pingMsg" {
		t.Errorf("expected pingMsg, got %v", ping.String())
	}
	if ref := "hello world"; ref != ping.Repr() {
		t.Errorf("expected %v, got %v", ref, ping.Repr())
	}
}

func BenchmarkPingEncode(b *testing.B) {
	out := make([]byte, 1024)
	ping := newPing("hello world")
	for i := 0; i < b.N; i++ {
		ping.Encode(out)
	}
}

func BenchmarkPingDecode(b *testing.B) {
	out := make([]byte, 1024)
	ref := newPing("made in india")
	n := ref.Encode(out)
	ping := newPing("")
	for i := 0; i < b.N; i++ {
		ping.Decode(out[:n])
	}
}
