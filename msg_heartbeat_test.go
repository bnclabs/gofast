package gofast

import "testing"
import "bytes"
import "reflect"
import "fmt"

var _ = fmt.Sprintf("dummy")

func TestHbEncode(t *testing.T) {
	out := make([]byte, 1024)
	ref := []byte{159, 26, 0, 152, 150, 128, 255}
	hb := NewHeartbeat(10000000)
	if n := hb.Encode(out); bytes.Compare(ref, out[:n]) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func TestHbDecode(t *testing.T) {
	out := make([]byte, 1024)
	ref := NewHeartbeat(10000000)
	n := ref.Encode(out)
	hb := &Heartbeat{}
	hb.Decode(out[:n])
	if !reflect.DeepEqual(ref, hb) {
		t.Errorf("expected %v, got %v", ref, hb)
	}
}

func BenchmarkHbEncode(b *testing.B) {
	out := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		hb := NewHeartbeat(10000000)
		hb.Encode(out)
		hbpool.Put(hb)
	}
}

func BenchmarkHbDecode(b *testing.B) {
	out := make([]byte, 1024)
	ref := NewHeartbeat(10000000)
	n := ref.Encode(out)
	hb := &Heartbeat{}
	for i := 0; i < b.N; i++ {
		hb.Decode(out[:n])
	}
}