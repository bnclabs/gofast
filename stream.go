package gofast

// Stream for a newly started stream on the transport. Refer to
// Stream() method on the transport.
type Stream struct {
	transport *Transport
	Rxch      chan Message
	opaque    uint64
	remote    bool
	out       []byte
	data      []byte
	tagout    []byte
}

// constructor used for remote streams.
func (t *Transport) newstream(opaque uint64, remote bool) *Stream {
	stream := t.fromrxstrm()
	//fmsg := "%v ##%d(remote:%v) stream created ...\n"
	//log.Verbosef(fmsg, t.logprefix, opaque, remote)
	// reset all fields (it is coming from a pool)
	stream.transport, stream.remote, stream.opaque = t, remote, opaque
	stream.Rxch = nil
	return stream
}

func (t *Transport) getstream(ch chan Message) *Stream { // called only be tx.
	stream := <-t.strmpool
	stream.Rxch = ch
	t.putch(t.rxch, rxpacket{stream: stream})
	return stream
}

func (t *Transport) putstream(opaque uint64, stream *Stream, tellrx bool) {
	// Rxch could also be closed when transport is closed...
	// Rxch could also be nil in case of post...
	defer func() {
		if r := recover(); r != nil {
			fmsg := "%v ##%v putstream recovered: %v\n"
			log.Infof(fmsg, t.logprefix, opaque, r)
		}
	}()
	if stream == nil {
		log.Errorf("%v ##%v unkown stream\n", t.logprefix, opaque)
		return
	}
	if stream.Rxch != nil {
		close(stream.Rxch)
	}
	stream.Rxch = nil
	if tellrx {
		t.putch(t.rxch, rxpacket{stream: stream})
	}
}

// Response to a request, to batch the response pass flush as false.
func (s *Stream) Response(msg Message, flush bool) error {
	n := s.transport.response(msg, s, s.out)
	return s.transport.txasync(s.out[:n], flush)
}

// Stream a single message, to batch the message pass flush as false.
func (s *Stream) Stream(msg Message, flush bool) (err error) {
	n := s.transport.stream(msg, s, s.out)
	if err = s.transport.txasync(s.out[:n], flush); err != nil {
		s.transport.putstream(s.opaque, s, true /*tellrx*/)
	}
	return
}

// Close this stream.
func (s *Stream) Close() error {
	n := s.transport.finish(s, s.out)
	s.transport.putstream(s.opaque, s, true /*tellrx*/)
	return s.transport.txasync(s.out[:n], true /*flush*/)
}

// Transport return the underlying transport carrying this stream.
func (s *Stream) Transport() *Transport {
	return s.transport
}
