package gofast

// not thread safe
type Stream struct {
	transport *Transport
	Rxch      chan Message
	opaque    uint64
	blueprint map[uint64]interface{}
}

func (t *Transport) newstream(opaque uint64) *Stream {
	return &Stream{transport: t, opaque: opaque}
}

func (t *Transport) getstream(rxch chan Message) *Stream {
	stream := <-t.streams
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

func (s *Stream) Send(msg Message) error {
	return s.transport.stream(s, msg)
}

func (s *Stream) SendAndClose(msg Message) error {
	if err := s.transport.stream(s, msg); err != nil {
		return err
	}
	return s.transport.finish(s)
}

func (s *Stream) Close() error {
	return s.transport.finish(s)
}
