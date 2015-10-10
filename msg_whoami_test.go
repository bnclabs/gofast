package gofast

import "testing"
import "bytes"
import "reflect"
import "fmt"

var _ = fmt.Sprintf("dummy")

func TestWaiEncode(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config, conn := newconfig("testtransport", st, end), newTestConnection()
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(conn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}

	out := make([]byte, 1024)
	ref := []byte{
		159, 109, 116, 101, 115, 116, 116, 114, 97, 110, 115, 112, 111, 114,
		116, 1, 26, 0, 160, 0, 0, 96, 255}
	wai := NewWhoami(trans)
	if n := wai.Encode(out); bytes.Compare(ref, out[:n]) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func TestWaiDecode(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config, conn := newconfig("testtransport", st, end), newTestConnection()
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(conn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}

	out := make([]byte, 1024)
	ref := NewWhoami(trans)
	n := ref.Encode(out)
	wai := &Whoami{}
	wai.Decode(out[:n])
	wai.version = int(wai.version.(uint64))
	if !reflect.DeepEqual(ref, wai) {
		t.Errorf("expected %#v, got %#v", ref, wai)
	}
}

func BenchmarkWaiEncode(b *testing.B) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config, conn := newconfig("testtransport", st, end), newTestConnection()
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(conn, testVersion(1), nil, config)
	if err != nil {
		b.Error(err)
	}

	out := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		wai := NewWhoami(trans)
		wai.Encode(out)
		whoamipool.Put(wai)
	}
}

func BenchmarkWaiDecode(b *testing.B) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config, conn := newconfig("testtransport", st, end), newTestConnection()
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(conn, testVersion(1), nil, config)
	if err != nil {
		b.Error(err)
	}

	out := make([]byte, 1024)
	ref := NewWhoami(trans)
	ref.tags = "gzip"
	n := ref.Encode(out)
	wai := &Whoami{}
	for i := 0; i < b.N; i++ {
		wai.Decode(out[:n])
	}
}
