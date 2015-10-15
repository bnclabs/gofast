package gofast

import "testing"
import "bytes"
import "reflect"
import "fmt"

var _ = fmt.Sprintf("dummy")

func TestPingEncode(t *testing.T) {
	out := make([]byte, 1024)
	ref := []byte{
		159, 107, 104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 255,
	}
	ping := NewPing("hello world")
	if n := ping.Encode(out); bytes.Compare(ref, out[:n]) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func TestPingDecode(t *testing.T) {
	out := make([]byte, 1024)
	ref := NewPing("made in india")
	n := ref.Encode(out)
	ping := &Ping{}
	ping.Decode(out[:n])
	if !reflect.DeepEqual(ref, ping) {
		t.Errorf("expected %v, got %v", ref, ping)
	}
}

func TestPingMisc(t *testing.T) {
	ping := NewPing("hello world")
	if ping.String() != "Ping" {
		t.Errorf("expected Ping, got %v", ping.String())
	}
	if ref := "hello world"; ref != ping.Repr() {
		t.Errorf("expected %v, got %v", ref, ping.Repr())
	}
}

func BenchmarkPingEncode(b *testing.B) {
	out := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		ping := NewPing("hello world")
		ping.Encode(out)
		pingpool.Put(ping)
	}
}

func BenchmarkPingDecode(b *testing.B) {
	out := make([]byte, 1024)
	ref := NewPing("made in india")
	n := ref.Encode(out)
	for i := 0; i < b.N; i++ {
		ping := pingpool.Get().(*Ping)
		ping.Decode(out[:n])
		pingpool.Put(ping)
	}
}
