package gofast

import "sync/atomic"
import "fmt"
import "runtime/debug"

type rxpacket struct {
	stream  *Stream
	msg     BinMessage
	opaque  uint64
	post    bool
	request bool
	start   bool
	strmsg  bool
	finish  bool
}

func (t *Transport) syncRx() {
	chansize := t.chansize
	livestreams := make(map[uint64]*Stream)
	defer func() {
		if r := recover(); r != nil {
			errorf("syncRx() panic: %v\n", r)
			errorf("\n%s", getStackTrace(2, debug.Stack()))
			go t.Close()
		}
		// unblock routines waiting on this stream
		for _, stream := range livestreams {
			if stream.rxcallb != nil {
				stream.rxcallb(BinMessage{}, false)
			}
		}
		t.flushrxch()
	}()

	streamupdate := func(stream *Stream) {
		_, ok := livestreams[stream.opaque]
		if ok && stream.rxcallb == nil {
			//TODO: Issue #2, remove or prevent value escape to heap
			//fmsg := "%v ##%d stream closed ...\n"
			//debugf(fmsg, t.logprefix, stream.opaque)
			delete(livestreams, stream.opaque)
			if stream.remote == false {
				t.pStrms <- stream // don't collect remote streams
			}
		} else if stream.rxcallb != nil {
			//TODO: Issue #2, remove or prevent value escape to heap
			//fmsg := "%v ##%d stream started ...\n"
			//verbosef(fmsg, t.logprefix, stream.opaque)
			livestreams[stream.opaque] = stream
		}
	}

	handlepkt := func(rxpkt rxpacket) {
		stream, streamok := livestreams[rxpkt.opaque]

		if streamok && rxpkt.finish {
			//TODO: Issue #2, remove or prevent value escape to heap
			//fmsg := "%v ##%d stream closed by remote ...\n"
			//debugf(fmsg, t.logprefix, stream.opaque)
			if stream.rxcallb != nil {
				stream.rxcallb(BinMessage{}, false)
			}
			t.putstream(rxpkt.opaque, stream, false /*tellrx*/)
			delete(livestreams, rxpkt.opaque)
			atomic.AddUint64(&t.nRxfin, 1)
			return

		} else if rxpkt.finish {
			//TODO: Issue #2, remove or prevent value escape to heap
			//fmsg := "%v ##%d unknown stream-fin from remote ...\n"
			//debugf(fmsg, t.logprefix, rxpkt.opaque)
			atomic.AddUint64(&t.nMdrops, 1)
			return
		}
		//TODO: Issue #2, remove or prevent value escape to heap
		//fmsg := "%v received msg %#v streamok:%v\n"
		//debugf(fmsg, t.logprefix, rxpkt.msg.ID, streamok)
		if streamok == false { // post, request, stream-start
			if rxpkt.post {
				t.requestCallback(nil /*stream*/, rxpkt.msg)
				atomic.AddUint64(&t.nRxpost, 1)
			} else if rxpkt.request {
				stream = t.newremotestream(rxpkt.opaque)
				t.requestCallback(stream, rxpkt.msg)
				atomic.AddUint64(&t.nRxreq, 1)
			} else if rxpkt.start { // stream
				stream = t.newremotestream(rxpkt.opaque)
				stream.rxcallb = t.requestCallback(stream, rxpkt.msg)
				livestreams[stream.opaque] = stream
				atomic.AddUint64(&t.nRxstart, 1)
			} else { // message for a closed stream.
				atomic.AddUint64(&t.nMdrops, 1)
			}
			return
		}

		// response and stream - finish is already handled above
		if stream.rxcallb != nil {
			if rxpkt.request {
				stream.rxcallb(rxpkt.msg, false)
			} else {
				stream.rxcallb(rxpkt.msg, true)
			}
		}

		if streamok && rxpkt.request { //means response
			atomic.AddUint64(&t.nRxresp, 1)
		} else if rxpkt.strmsg {
			atomic.AddUint64(&t.nRxstream, 1)
		} else {
			fmsg := "%v duplicate rxpkt ##%d for stream ##%d %#v ...\n"
			warnf(fmsg, t.logprefix, rxpkt.opaque, stream.opaque, rxpkt)
			atomic.AddUint64(&t.nMdrops, 1)
		}
	}

	go t.doRx()

	fmsg := "%v syncRx(chansize:%v) started ...\n"
	infof(fmsg, t.logprefix, chansize)
loop:
	for {
		select {
		case rxpkt := <-t.rxch:
			if rxpkt.stream != nil {
				streamupdate(rxpkt.stream)
				rxpkt.stream = nil
			} else {
				handlepkt(rxpkt)
				if rxpkt.msg.Data != nil {
					t.putdata(rxpkt.msg.Data)
					rxpkt.msg.Data = nil
				}
				atomic.AddUint64(&t.nRx, 1)
			}
		case <-t.killch:
			break loop
		}
	}

	infof("%v syncRx() ... stopped\n", t.logprefix)
}

func (t *Transport) putch(ch chan rxpacket, val rxpacket) bool {
	select {
	case ch <- val:
		return true
	case <-t.killch:
		return false
	}
}

func (t *Transport) flushrxch() {
	// flush out pending messages from rxch
	for {
		select {
		case rxpkt := <-t.rxch:
			if rxpkt.stream != nil && rxpkt.stream.rxcallb != nil {
				rxpkt.stream.rxcallb(BinMessage{}, false)
			}
		default:
			return
		}
	}
}

func (r *rxpacket) String() string {
	return fmt.Sprintf("##%d %v %v %v", r.opaque, r.request, r.start, r.finish)
}
