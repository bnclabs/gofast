package gofast

// Stream for a newly started stream on the transport.
type Stream struct {
	transport *Transport
	Rxch      chan Message
	opaque    uint64
	remote    bool
}

func (t *Transport) newstream(opaque uint64, remote bool) *Stream {
	fmsg := "%v ##%d(remote:%v) stream started ...\n"
	log.Verbosef(fmsg, t.logprefix, opaque, remote)
	return &Stream{transport: t, remote: remote, opaque: opaque, Rxch: nil}
}

func (t *Transport) getstream(ch chan Message) *Stream {
	stream := <-t.strmpool
	stream.Rxch = ch
	t.putch(t.rxch, stream)
	return stream
}

func (t *Transport) putstream(opaque uint64, stream *Stream, tellrx bool) {
	func() {
		// Rxch could also be closed when transport is closed...
		// Rxch could also be nil in case of post...
		defer func() { recover() }()
	}()
	if stream == nil {
		log.Errorf("%v ##%v unkown stream\n", t.logprefix, opaque)
		return
	}
	if stream.Rxch != nil {
		close(stream.Rxch)
	}
	stream.Rxch = nil
	if stream.remote == false { // reclaim if local stream
		t.strmpool <- stream
	}
	if tellrx {
		t.putch(t.rxch, stream)
	}
}

// Response to a request.
func (s *Stream) Response(msg Message) error {
	out := s.transport.pktpool.Get().([]byte)
	defer s.transport.pktpool.Put(out)
	defer s.transport.putstream(s.opaque, s, true /*tellrx*/)

	n := s.transport.response(msg, s, out)
	return s.transport.tx(out[:n], false)
}

// Stream a message.
func (s *Stream) Stream(msg Message) (err error) {
	out := s.transport.pktpool.Get().([]byte)
	defer s.transport.pktpool.Put(out)
	defer func() {
		if err != nil {
			s.transport.putstream(s.opaque, s, true /*tellrx*/)
		}
	}()
	n := s.transport.stream(msg, s, out)
	err = s.transport.tx(out[:n], false)
	return
}

// Close the stream.
func (s *Stream) Close() error {
	out := s.transport.pktpool.Get().([]byte)
	defer s.transport.pktpool.Put(out)
	defer s.transport.putstream(s.opaque, s, true /*tellrx*/)

	n := s.transport.finish(s, out)
	return s.transport.tx(out[:n], false)
}

// Transport return the underlying transport carrying this stream.
func (s *Stream) Transport() *Transport {
	return s.transport
}
