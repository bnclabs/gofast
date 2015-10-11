package gofast

import "testing"
import "fmt"
import "bytes"

var _ = fmt.Sprintf("dummy")

//func (t *Transport) post(msg Message, stream *Stream, out []byte) (n int) {
//	n = tag2cbor(tagCborPrefix, out)      // prefix
//	n += t.framepkt(stream, msg, out[n:]) // packet
//	return n
//}
//
//// | 0xd9f7 | 0x91 | packet |
//func (t *Transport) request(msg Message, stream *Stream, out []byte) (n int) {
//	n = tag2cbor(tagCborPrefix, out) // prefix
//	out[n] = 0x91                    // 0x91
//	n += 1
//	n += t.framepkt(stream, msg, out[n:]) // packet
//	return n
//}
//
//// | 0xd9f7 | 0x91 | packet |
//func (t *Transport) response(msg Message, stream *Stream, out []byte) (n int) {
//	n = tag2cbor(tagCborPrefix, out) // prefix
//	out[n] = 0x91                    // 0x91
//	n += 1
//	n += t.framepkt(stream, msg, out[n:]) // packet
//	return n
//}
//
//// | 0xd9f7 | 0x9f | packet1 |
//func (t *Transport) start(msg Message, stream *Stream, out []byte) (n int) {
//	n = tag2cbor(tagCborPrefix, out)      // prefix
//	n += arrayStart(out[n:])              // 0x9f
//	n += t.framepkt(stream, msg, out[n:]) // packet
//	return n
//}
//
//// | 0xd9f7 | packet2 |
//func (t *Transport) stream(stream *Stream, msg Message, out []byte) (n int) {
//	n = tag2cbor(tagCborPrefix, out)      // prefix
//	n += t.framepkt(stream, msg, out[n:]) // packet
//	return n
//}
//
//// | 0xd9f7 | packetN | 0xff |
//func (t *Transport) finish(stream *Stream, out []byte) (n int) {
//	var scratch [16]byte
//	n = tag2cbor(tagCborPrefix, out)         // prefix
//	m := tag2cbor(stream.opaque, scratch[:]) // tag-opaque
//	scratch[m] = 0xff                        // 0xff (payload)
//	m += 1
//	n += value2cbor(scratch[:m], out[n:]) // packet
//	return n
//}

func TestFramePkt(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}

	ref := []byte{
		88, 38, 217, 1, 0, 88, 33, 216, 37, 191, 24, 39, 2, 24, 40, 87, 159, 109,
		116, 101, 115, 116, 116, 114, 97, 110, 115, 112, 111, 114, 116, 1, 26, 0,
		160, 0, 0, 96, 255, 255}
	stream := trans.getstream(nil)
	wai := NewWhoami(trans)
	out := make([]byte, 1024)
	n := trans.framepkt(stream, wai, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func BenchmarkFramePkt(b *testing.B) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		b.Error(err)
	}
	stream := trans.getstream(nil)
	ping := NewPing("hello world")
	out := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trans.framepkt(stream, ping, out)
	}
}
