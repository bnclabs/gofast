package gofast

import "testing"
import "bytes"
import "reflect"

func TestPingEncode(t *testing.T) {
	out := make([]byte, 1024)
	ref := []byte{
		0, 11, 104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100,
	}
	ping := newPing("hello world")
	if out := ping.Encode(out); bytes.Compare(ref, out) != 0 {
		t.Errorf("expected %v, got %v", ref, out)
	}
}

func TestPingDecode(t *testing.T) {
	out := make([]byte, 1024)
	ref := newPing("made in india")
	out = ref.Encode(out)
	ping := &pingMsg{}
	ping.Decode(out)
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
	out = ref.Encode(out)
	ping := newPing("")
	for i := 0; i < b.N; i++ {
		ping.Decode(out)
	}
}
