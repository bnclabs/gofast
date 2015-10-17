package gofast

// Stream for a newly started stream on the transport.
type Stream struct {
	transport *Transport
	Rxch      chan Message
	opaque    uint64
	remote    bool
}

func (t *Transport) newstream(opaque uint64, remote bool) *Stream {
	fmsg := "%v ##%d(remote:%v) stream created ...\n"
	log.Verbosef(fmsg, t.logprefix, opaque, remote)
	return &Stream{transport: t, remote: remote, opaque: opaque, Rxch: nil}
}

func (t *Transport) getstream(ch chan Message) *Stream {
	stream := <-t.strmpool
	stream.Rxch = ch
	t.putch(t.rxch, stream)
	return stream
}

func (t *Transport) putstream(stream *Stream, tellrx bool) {
	func() {
		// Rxch could also be closed when transport is closed...
		// Rxch could also be nil in case of post...
		defer func() { recover() }()
	}()
	if stream == nil {
		log.Errorf("%v ##%v unkown stream\n", t.logprefix, stream.opaque)
		return
	}
	if stream.Rxch != nil {
		close(stream.Rxch)
	}
	stream.Rxch = nil
	if tellrx {
		t.putch(t.rxch, stream)
	}
}

// Response to a request.
func (s *Stream) Response(msg Message, flush bool) error {
	obj := s.transport.pktpool.Get()
	defer s.transport.pktpool.Put(obj)

	out := obj.([]byte)
	n := s.transport.response(msg, s, out)
	s.transport.putstream(s, true /*tellrx*/)
	return s.transport.tx(out[:n], flush)
}

// Stream a message.
func (s *Stream) Stream(msg Message, flush bool) (err error) {
	obj := s.transport.pktpool.Get()
	defer s.transport.pktpool.Put(obj)

	out := obj.([]byte)
	n := s.transport.stream(msg, s, out)
	if err = s.transport.tx(out[:n], flush); err != nil {
		s.transport.putstream(s, true /*tellrx*/)
	}
	return
}

// Close the stream.
func (s *Stream) Close() error {
	obj := s.transport.pktpool.Get()
	defer s.transport.pktpool.Put(obj)

	out := obj.([]byte)
	n := s.transport.finish(s, out)
	s.transport.putstream(s, true /*tellrx*/)
	return s.transport.tx(out[:n], true /*flush*/)
}

// Transport return the underlying transport carrying this stream.
func (s *Stream) Transport() *Transport {
	return s.transport
}
