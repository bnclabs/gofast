package gofast

import "testing"
import "fmt"
import "bytes"

var _ = fmt.Sprintf("dummy")

func TestPost(t *testing.T) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 88, 31, 217, 1, 12, 88, 26, 216, 37, 191, 24, 39,
		2, 24, 40, 80, 159, 70, 99, 108, 105, 101, 110, 116, 1, 26, 0,
		160, 0, 0, 64, 255, 255,
	}
	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(transc)
	n := transc.post(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestRequest(t *testing.T) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 145, 88, 31, 217, 1, 12, 88, 26, 216, 37, 191, 24,
		39, 2, 24, 40, 80, 159, 70, 99, 108, 105, 101, 110, 116, 1, 26,
		0, 160, 0, 0, 64, 255, 255,
	}
	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(transc)
	n := transc.request(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestResponse(t *testing.T) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 145, 88, 31, 217, 1, 12, 88, 26, 216, 37, 191, 24, 39,
		2, 24, 40, 80, 159, 70, 99, 108, 105, 101, 110, 116, 1, 26, 0, 160,
		0, 0, 64, 255, 255,
	}
	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(transc)
	n := transc.response(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestStart(t *testing.T) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 159, 88, 31, 217, 1, 12, 88, 26, 216, 37, 191, 24, 39,
		2, 24, 40, 80, 159, 70, 99, 108, 105, 101, 110, 116, 1, 26, 0, 160,
		0, 0, 64, 255, 255,
	}
	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(transc)
	n := transc.start(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestStream(t *testing.T) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 88, 31, 217, 1, 12, 88, 26, 216, 37, 191, 24, 39,
		2, 24, 40, 80, 159, 70, 99, 108, 105, 101, 110, 116, 1, 26, 0,
		160, 0, 0, 64, 255, 255,
	}
	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(transc)
	n := transc.stream(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestFinish(t *testing.T) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	ref := []byte{217, 217, 247, 68, 217, 1, 12, 255}
	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	n := transc.finish(stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestFramePkt(t *testing.T) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	ref := []byte{
		88, 31, 217, 1, 12, 88, 26, 216, 37, 191, 24, 39, 2, 24, 40, 80,
		159, 70, 99, 108, 105, 101, 110, 116, 1, 26, 0, 160, 0, 0, 64,
		255, 255,
	}
	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(transc)
	n := transc.framepkt(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkPostPkt(b *testing.B) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.post(wai, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkRequestPkt(b *testing.B) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.request(wai, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkResponsePkt(b *testing.B) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.response(wai, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkStartPkt(b *testing.B) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.start(wai, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkStreamPkt(b *testing.B) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.stream(wai, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkFinishPkt(b *testing.B) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.finish(stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkFramePkt(b *testing.B) {
	// init server
	lis, serverch := newServer("")
	// init client
	transc := newClient("").Handshake()
	transv := <-serverch

	stream := transc.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.framepkt(wai, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}
