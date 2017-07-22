package gofast

import "testing"
import "bytes"
import "reflect"

func TestHbEncode(t *testing.T) {
	out := make([]byte, 1024)
	ref := []byte{0, 0, 0, 0, 0, 152, 150, 128}
	hb := newHeartbeat(10000000)
	if out = hb.Encode(out); bytes.Compare(ref, out) != 0 {
		t.Errorf("expected %v, got %v", ref, out)
	}
}

func TestHbDecode(t *testing.T) {
	out := make([]byte, 1024)
	ref := newHeartbeat(10000000)
	out = ref.Encode(out)
	hb := &heartbeatMsg{}
	hb.Decode(out)
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
	out = ref.Encode(out)
	hb := newHeartbeat(1000000)
	for i := 0; i < b.N; i++ {
		hb.Decode(out)
	}
}
