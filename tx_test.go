package gofast

import "testing"
import "fmt"
import "bytes"

var _ = fmt.Sprintf("dummy")

func TestPost(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 198, 88, 29, 217, 1, 12, 88, 24, 216, 37, 191,
		24, 39, 2, 24, 40, 78, 159, 70, 99, 108, 105, 101, 110, 116,
		1, 25, 2, 0, 64, 255, 255,
	}
	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	wai := newWhoami(transc)
	n := transc.post(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestRequest(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 129, 88, 29, 217, 1, 12, 88, 24, 216, 37, 191,
		24, 39, 2, 24, 40, 78, 159, 70, 99, 108, 105, 101, 110, 116,
		1, 25, 2, 0, 64, 255, 255,
	}
	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	wai := newWhoami(transc)
	n := transc.request(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestResponse(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 129, 88, 29, 217, 1, 12, 88, 24, 216, 37, 191,
		24, 39, 2, 24, 40, 78, 159, 70, 99, 108, 105, 101, 110, 116,
		1, 25, 2, 0, 64, 255, 255,
	}
	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	wai := newWhoami(transc)
	n := transc.response(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestStart(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 159, 88, 29, 217, 1, 12, 88, 24, 216, 37, 191, 24,
		39, 2, 24, 40, 78, 159, 70, 99, 108, 105, 101, 110, 116, 1, 25, 2,
		0, 64, 255, 255,
	}
	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	wai := newWhoami(transc)
	n := transc.start(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestStream(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 199, 88, 29, 217, 1, 12, 88, 24, 216, 37, 191, 24, 39,
		2, 24, 40, 78, 159, 70, 99, 108, 105, 101, 110, 116, 1, 25, 2, 0,
		64, 255, 255,
	}
	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	wai := newWhoami(transc)
	n := transc.stream(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func TestFinish(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	ref := []byte{217, 217, 247, 200, 68, 217, 1, 12, 255}
	stream := transc.getlocalstream(nil, false)
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
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	ref := []byte{
		88, 29, 217, 1, 12, 88, 24, 216, 37, 191, 24, 39, 2, 24, 40,
		78, 159, 70, 99, 108, 105, 101, 110, 116, 1, 25, 2, 0, 64,
		255, 255,
	}
	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	wai := newWhoami(transc)
	n := transc.framepkt(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkPostPkt(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	msg := newPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.post(msg, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkRequestPkt(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	msg := newPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.request(msg, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkResponsePkt(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	msg := newPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.response(msg, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkStartPkt(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	msg := newPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.start(msg, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkStreamPkt(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	msg := newPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.stream(msg, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkFinishPkt(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	stream := transc.getlocalstream(nil, false)
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
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	stream := transc.getlocalstream(nil, false)
	out := make([]byte, 1024)
	msg := newPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transc.framepkt(msg, stream, out)
	}
	lis.Close()
	transc.Close()
	transv.Close()
}
