package gofast

// Stream for every stream started on the transport.
type Stream struct {
	transport *Transport
	Rxch      chan Message
	opaque    uint64
	remote    bool
	blueprint map[uint64]interface{}
}

func (t *Transport) newstream(opaque uint64) *Stream {
	return &Stream{transport: t, remote: false, opaque: opaque}
}

func (t *Transport) getstream(ch chan Message) *Stream {
	stream := <-t.streams
	stream.Rxch = ch
	select {
	case t.rxch <- stream:
	case <-t.killch:
	}
	return stream
}

func (t *Transport) putstream(stream *Stream) {
	func() {
		// Rxch could also be closed when transport is closed...
		defer func() { recover() }()
		close(stream.Rxch)
	}()
	stream.Rxch = nil
	select {
	case t.rxch <- stream: // clean from syncrx book keeping
		if stream.remote == false { // reclaim if local stream
			t.streams <- stream
		}
	case <-t.killch:
	}
}

// Send a message on the stream.
func (s *Stream) Send(msg Message) error {
	return s.transport.stream(s, msg)
}

// Send the last message on the stream and close the stream.
func (s *Stream) SendAndClose(msg Message) error {
	if err := s.transport.stream(s, msg); err != nil {
		return err
	}
	return s.transport.finish(s)
}

// Close the stream.
func (s *Stream) Close() error {
	return s.transport.finish(s)
}

// Transport carrying this stream.
func (s *Stream) Transport() *Transport {
	return s.transport
}
