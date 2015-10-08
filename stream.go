package gofast

// Stream for every stream started on the transport.
type Stream struct {
	transport *Transport
	Rxch      chan Message
	opaque    uint64
	remote    bool
}

func (t *Transport) newstream(opaque uint64, remote bool) *Stream {
	fmsg := "%v ##%d(remote:%v) stream started ...\n"
	log.Debugf(fmsg, t.logprefix, opaque, remote)
	return &Stream{transport: t, remote: remote, opaque: opaque, Rxch: nil}
}

func (t *Transport) getstream(ch chan Message) *Stream {
	stream := <-t.streams
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
		log.Errorf("%v unkown stream ##%v\n", t.logprefix, opaque)
		return
	}
	close(stream.Rxch)
	stream.Rxch = nil
	if stream.remote == false { // reclaim if local stream
		t.streams <- stream
	}
	if tellrx {
		t.putch(t.rxch, stream)
	}
}

// Response a request.
func (s *Stream) Response(msg Message) error {
	return s.transport.stream(s, msg)
}

// Stream a message on the stream.
func (s *Stream) Stream(msg Message) error {
	return s.transport.stream(s, msg)
}

// Close the stream.
func (s *Stream) Close() error {
	return s.transport.finish(s)
}

// Transport carrying this stream.
func (s *Stream) Transport() *Transport {
	return s.transport
}
