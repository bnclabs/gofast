package gofast

import "sync"

// not thread safe
type Stream struct {
	t         *Transport
	Rxch      chan Message
	opaque    uint64
	blueprint map[uint64]interface{}
}

func (t *Transport) newstream(opaque uint64) *Stream {
	return &Stream{t: t, opaque: opaque}
}

func (t *Transport) getstream(rxch chan Message) *Stream {
	stream <- t.streams
	stream.Rxch = rxch
	t.rxch <- stream
	return stream
}

func (t *Transport) putstream(stream *Stream) {
	close(stream.Rxch)
	stream.Rxch = nil
	t.rxch <- stream // clean from syncrx book keeping
	t.streams <- stream
}

func (s *Stream) Send(msg Message) {
}

func (s *Stream) SendAndClose(msg Message) {
}

func (s *Stream) Close() {
}
