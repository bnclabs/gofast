package gofast

import "testing"
import "fmt"
import "bytes"
import "reflect"

var _ = fmt.Sprintf("dummy")

func TestReadtagp(t *testing.T) {
	// read tag and its payload
	payload := []byte{
		216, 37, 191, 24, 39, 2, 24, 40, 87, 159, 109, 116, 101, 115,
		116, 116, 114, 97, 110, 115, 112, 111, 114, 116, 1, 26, 0,
		160, 0, 0, 96, 255, 255}
	tag, bs := readtp(payload)
	// read tagMsg and its payload
	if tag != tagMsg {
		t.Errorf("expected %v, got %v", tagMsg, tag)
	} else if bytes.Compare(payload[2:], bs) != 0 {
		t.Errorf("expected %v, got %v", payload[2:], bs)
	}
}

func TestUnmessage(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer(addr, "")      // init server
	transc := newClient(addr, "").Handshake() // init client
	transv := <-serverch

	// read tag and its payload
	payload := []byte{
		216, 37, 191, 24, 39, 2, 24, 40, 78,
		159, 70, 99, 108, 105, 101, 110, 116, 1, 25, 2, 0, 64, 255,
		255}

	_, bs := readtp(payload)
	// unmessage
	ref := newWhoami(transc)
	msg := transc.unmessage(100, bs).(*whoamiMsg)
	if !reflect.DeepEqual(ref, msg) {
		t.Errorf("expected %#v, got %#v", ref, msg)
	}

	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkReadtagp(b *testing.B) {
	// read tag and its payload
	payload := []byte{
		216, 37, 191, 24, 39, 2, 24, 40, 87,
		159, 109, 116, 101, 115, 116, 116, 114, 97, 110, 115, 112, 111,
		114, 116, 1, 26, 0, 160, 0, 0, 96, 255,
		255}
	for i := 0; i < b.N; i++ {
		readtp(payload)
	}
}

func BenchmarkUnmessage(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer(addr, "")      // init server
	transc := newClient(addr, "").Handshake() // init client
	transv := <-serverch

	// read tag and its payload
	payload := []byte{
		216, 37, 191, 24, 39, 2, 24, 40, 78,
		159, 70, 99, 108, 105, 101, 110, 116, 1, 25, 2, 0, 64, 255,
		255}
	_, bs := readtp(payload)
	b.ResetTimer()
	// unmessage
	for i := 0; i < b.N; i++ {
		transc.unmessage(100, bs)
	}

	lis.Close()
	transc.Close()
	transv.Close()
}
