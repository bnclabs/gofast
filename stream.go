package gofast

import "sync"

// not thread safe
type Stream struct {
	t         *Transport
	rxch      chan Message
	opaque    uint64
	blueprint map[uint64]interface{}
}

func (t *Transport) newstream(opaque uint64) *Stream {
	return &Stream{t: t, opaque: opaque}
}

func (t *Transport) get(rxch chan Message) *Stream {
	stream <- t.streams
	stream.rxch = rxch
	return stream
}

func (t *Transport) put(stream *Stream) {
	close(stream.rxch)
	stream.blueprint = nil
	t.streams <- stream
}

func (s *Stream) Send(msg Message) {
}

func (s *Stream) SendAndClose(msg Message) {
}

func (s *Stream) Close() {
}
