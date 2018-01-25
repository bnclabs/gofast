package gofast

import "sync/atomic"

import "github.com/bnclabs/golog"

// Stream for a newly started stream on the transport. Refer to
// Stream() method on the transport.
type Stream struct {
	transport         *Transport
	rxcallb           StreamCallback
	opaque            uint64
	remote            bool
	out, data, tagout []byte
}

// constructor used for remote streams.
func (t *Transport) newremotestream(opaque uint64) *Stream {
	stream := t.fromrxstrm()

	//TODO: Issue #2, remove or prevent value escape to heap
	//fmsg := "%v ##%d(remote:%v) stream created ...\n"
	//log.Verbosef(fmsg, t.logprefix, opaque, remote)

	// reset all fields (it is coming from a pool)
	stream.transport, stream.remote, stream.opaque = t, true, opaque
	stream.rxcallb = nil
	return stream
}

// called only be tx.
func (t *Transport) getlocalstream(tellrx bool, rxcallb StreamCallback) *Stream {
	stream := <-t.pStrms
	stream.rxcallb = rxcallb
	atomic.StoreUint64(&stream.opaque, stream.opaque)
	if tellrx {
		t.putch(t.rxch, rxpacket{stream: stream})
	}
	return stream
}

func (t *Transport) putstream(opaque uint64, stream *Stream, tellrx bool) {
	defer func() {
		if r := recover(); r != nil {
			fmsg := "%v ##%v putstream recovered: %v\n"
			log.Errorf(fmsg, t.logprefix, opaque, r)
		}
	}()

	if stream == nil {
		log.Errorf("%v ##%v unknown stream\n", t.logprefix, opaque)
		return
	}
	if tellrx {
		t.putch(t.rxch, rxpacket{stream: stream})
	} else if stream.remote == false {
		t.pStrms <- stream // don't collect remote streams
	}
}

// Response to a request, to batch the response pass flush as false.
func (s *Stream) Response(msg Message, flush bool) error {
	defer s.transport.pRxstrm.Put(s)
	n := s.transport.response(msg, s, s.out)
	return s.transport.txasync(s.out[:n], flush)
}

// Stream a single message, to batch the message pass flush as false.
func (s *Stream) Stream(msg Message, flush bool) (err error) {
	n := s.transport.stream(msg, s, s.out)
	return s.transport.txasync(s.out[:n], flush)
}

// Close this stream.
func (s *Stream) Close() error {
	n := s.transport.finish(s, s.out)
	return s.transport.txasync(s.out[:n], true /*flush*/)
}

// Transport return the underlying transport carrying this stream.
func (s *Stream) Transport() *Transport {
	return s.transport
}
