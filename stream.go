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
	stream := &Stream{t: t, opaque: opaque}
	stream.blueprint = t.cloneblueprint()
	return s
}

func (s *Stream) Send(msg Message) {
}

func (s *Stream) SendAndClose(msg Message) {
}

func (s *Stream) Close() {
}
