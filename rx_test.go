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
	laddr, raddr := "127.0.0.1:9998", "127.0.0.1:9999"
	ver := testVersion(1)
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(laddr, raddr, nil, false)
	config["tags"], config["log.level"] = "", "warn"
	trans, err := NewTransport(tconn, &ver, nil, config)
	if err != nil {
		t.Error(err)
	}

	// read tag and its payload
	payload := []byte{
		216, 37, 191, 24, 39, 2, 24, 40, 87, 159, 109, 116, 101, 115,
		116, 116, 114, 97, 110, 115, 112, 111, 114, 116, 1, 26, 0,
		160, 0, 0, 96, 255, 255}
	_, bs := readtp(payload)
	// unmessage
	ref := NewWhoami(trans)
	msg := trans.unmessage(100, bs).(*Whoami)
	if !reflect.DeepEqual(ref, msg) {
		t.Errorf("expected %v, got %v", ref, msg)
	}
}

func BenchmarkReadtagp(b *testing.B) {
	// read tag and its payload
	payload := []byte{
		216, 37, 191, 24, 39, 2, 24, 40, 87, 159, 109, 116, 101, 115,
		116, 116, 114, 97, 110, 115, 112, 111, 114, 116, 1, 26, 0,
		160, 0, 0, 96, 255, 255}
	for i := 0; i < b.N; i++ {
		readtp(payload)
	}
}

func BenchmarkUnmessage(b *testing.B) {
	laddr, raddr := "127.0.0.1:9998", "127.0.0.1:9999"
	ver := testVersion(1)
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(laddr, raddr, nil, false)
	config["tags"], config["log.level"] = "", "warn"
	trans, err := NewTransport(tconn, &ver, nil, config)
	if err != nil {
		b.Error(err)
	}

	// read tag and its payload
	payload := []byte{
		216, 37, 191, 24, 39, 2, 24, 40, 87, 159, 109, 116, 101, 115,
		116, 116, 114, 97, 110, 115, 112, 111, 114, 116, 1, 26, 0,
		160, 0, 0, 96, 255, 255}
	_, bs := readtp(payload)
	b.ResetTimer()
	// unmessage
	for i := 0; i < b.N; i++ {
		trans.unmessage(100, bs)
	}
}
