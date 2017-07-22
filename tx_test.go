package gofast

import "testing"
import "bytes"

func TestPost(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 198, 88, 35, 217, 1, 22, 88, 30, 216, 43, 191,
		216, 44, 25, 16, 2, 216, 45, 82,
		6, 99, 108, 105, 101, 110, 116, 1, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0,
		255,
	}
	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 129, 88, 35, 217, 1, 22, 88, 30, 216, 43, 191,
		216, 44, 25, 16, 2, 216, 45, 82,
		6, 99, 108, 105, 101, 110, 116, 1, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0,
		255,
	}
	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 129, 88, 35, 217, 1, 22, 88, 30, 216, 43, 191,
		216, 44, 25, 16, 2, 216, 45, 82,
		6, 99, 108, 105, 101, 110, 116, 1, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0,
		255,
	}
	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 159, 88, 35, 217, 1, 22, 88, 30, 216, 43, 191, 216,
		44, 25, 16, 2, 216, 45, 82,
		6, 99, 108, 105, 101, 110, 116, 1, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0,
		255,
	}
	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	ref := []byte{
		217, 217, 247, 199, 88, 35, 217, 1, 22, 88, 30, 216, 43, 191, 216, 44,
		25, 16, 2, 216, 45, 82,
		6, 99, 108, 105, 101, 110, 116, 1, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0,
		255,
	}
	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	ref := []byte{217, 217, 247, 200, 68, 217, 1, 22, 64, 255}
	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	ref := []byte{
		88, 35, 217, 1, 22, 88, 30, 216, 43, 191, 216, 44, 25, 16, 2, 216, 45,
		82, 6, 99, 108, 105, 101, 110, 116, 1, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0,
		255,
	}
	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	stream := transc.getlocalstream(false, nil)
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
	lis, serverch := newServer("server", addr, "") // init server
	transc := newClient("client", addr, "")
	if err := transc.Handshake(); err != nil { // init client
		panic(err)
	}
	transv := <-serverch

	stream := transc.getlocalstream(false, nil)
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
