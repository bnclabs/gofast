package gofast

import "testing"
import "bytes"
import "reflect"
import "fmt"

var _ = fmt.Sprintf("dummy")

func TestHbEncode(t *testing.T) {
	out := make([]byte, 1024)
	ref := []byte{159, 26, 0, 152, 150, 128, 255}
	hb := newHeartbeat(10000000)
	if n := hb.Encode(out); bytes.Compare(ref, out[:n]) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func TestHbDecode(t *testing.T) {
	out := make([]byte, 1024)
	ref := newHeartbeat(10000000)
	n := ref.Encode(out)
	hb := &heartbeatMsg{}
	hb.Decode(out[:n])
	if !reflect.DeepEqual(ref, hb) {
		t.Errorf("expected %v, got %v", ref, hb)
	}
}

func TestHbMisc(t *testing.T) {
	hb := newHeartbeat(10)
	if hb.String() != "heartbeatMsg" {
		t.Errorf("expected heartbeatMsg, got %v", hb.String())
	}
	if ref := "heartbeatMsg:10"; ref != hb.Repr() {
		t.Errorf("expected %v, got %v", ref, hb.Repr())
	}
}

func BenchmarkHbEncode(b *testing.B) {
	out := make([]byte, 1024)
	hb := newHeartbeat(10000000)
	for i := 0; i < b.N; i++ {
		hb.Encode(out)
	}
}

func BenchmarkHbDecode(b *testing.B) {
	out := make([]byte, 1024)
	ref := newHeartbeat(10000000)
	n := ref.Encode(out)
	hb := newHeartbeat(1000000)
	for i := 0; i < b.N; i++ {
		hb.Decode(out[:n])
	}
}
