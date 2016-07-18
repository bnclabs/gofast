package gofast

import "sync/atomic"
import "strings"
import "io"
import "fmt"
import "net"
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
			log.Errorf("syncRx() panic: %v\n", r)
			log.Errorf("\n%s", getStackTrace(2, debug.Stack()))
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
			//log.Debugf(fmsg, t.logprefix, stream.opaque)
			delete(livestreams, stream.opaque)
			if stream.remote == false {
				t.p_strms <- stream // don't collect remote streams
			}
		} else if stream.rxcallb != nil {
			//TODO: Issue #2, remove or prevent value escape to heap
			//fmsg := "%v ##%d stream started ...\n"
			//log.Verbosef(fmsg, t.logprefix, stream.opaque)
			livestreams[stream.opaque] = stream
		}
	}

	handlepkt := func(rxpkt rxpacket) {
		stream, streamok := livestreams[rxpkt.opaque]

		if streamok && rxpkt.finish {
			//TODO: Issue #2, remove or prevent value escape to heap
			//fmsg := "%v ##%d stream closed by remote ...\n"
			//log.Debugf(fmsg, t.logprefix, stream.opaque)
			if stream.rxcallb != nil {
				stream.rxcallb(BinMessage{}, false)
			}
			t.putstream(rxpkt.opaque, stream, false /*tellrx*/)
			delete(livestreams, rxpkt.opaque)
			atomic.AddUint64(&t.n_rxfin, 1)
			return

		} else if rxpkt.finish {
			//TODO: Issue #2, remove or prevent value escape to heap
			//fmsg := "%v ##%d unknown stream-fin from remote ...\n"
			//log.Debugf(fmsg, t.logprefix, rxpkt.opaque)
			atomic.AddUint64(&t.n_mdrops, 1)
			return
		}
		//TODO: Issue #2, remove or prevent value escape to heap
		//fmsg := "%v received msg %#v streamok:%v\n"
		//log.Debugf(fmsg, t.logprefix, rxpkt.msg.ID, streamok)
		if streamok == false { // post, request, stream-start
			if rxpkt.post {
				t.requestCallback(nil /*stream*/, rxpkt.msg)
				atomic.AddUint64(&t.n_rxpost, 1)
			} else if rxpkt.request {
				stream = t.newremotestream(rxpkt.opaque)
				t.requestCallback(stream, rxpkt.msg)
				atomic.AddUint64(&t.n_rxreq, 1)
			} else if rxpkt.start { // stream
				stream = t.newremotestream(rxpkt.opaque)
				stream.rxcallb = t.requestCallback(stream, rxpkt.msg)
				livestreams[stream.opaque] = stream
				atomic.AddUint64(&t.n_rxstart, 1)
			} else { // message for a closed stream.
				atomic.AddUint64(&t.n_mdrops, 1)
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
			if (cap(t.p_data) - len(t.p_data)) > 0 {
				t.p_data <- rxpkt.msg.Data
			}
			rxpkt.msg.Data = nil
		}

		if streamok && rxpkt.request { //means response
			atomic.AddUint64(&t.n_rxresp, 1)
		} else if rxpkt.strmsg {
			atomic.AddUint64(&t.n_rxstream, 1)
		} else {
			fmsg := "%v duplicate rxpkt ##%d for stream ##%d %#v ...\n"
			log.Warnf(fmsg, t.logprefix, rxpkt.opaque, stream.opaque, rxpkt)
			atomic.AddUint64(&t.n_mdrops, 1)
		}
	}

	go t.doRx()

	fmsg := "%v syncRx(chansize:%v) started ...\n"
	log.Infof(fmsg, t.logprefix, chansize)
loop:
	for {
		select {
		case rxpkt := <-t.rxch:
			if rxpkt.stream != nil {
				streamupdate(rxpkt.stream)
				rxpkt.stream = nil
			} else {
				handlepkt(rxpkt)
				atomic.AddUint64(&t.n_rx, 1)
			}
		case <-t.killch:
			break loop
		}
	}

	log.Infof("%v syncRx() ... stopped\n", t.logprefix)
}

func (t *Transport) doRx() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("doRx() panic: %v\n", r)
			log.Errorf("\n%s", getStackTrace(2, debug.Stack()))
		}
		go t.Close()
	}()

	log.Infof("%v doRx() started ...\n", t.logprefix)
	pad := make([]byte, 9)
	packet := make([]byte, t.buffersize)
	tagouts := make(map[uint64][]byte, t.buffersize)
	for _, factory := range tag_factory {
		tag, _, _ := factory(t, t.settings)
		tagouts[tag] = make([]byte, t.buffersize)
	}

	for {
		rxpkt, err := t.unframepkt(t.conn, pad, packet, tagouts)
		if err != nil {
			break
		}
		//TODO: Issue #2, remove or prevent value escape to heap
		//log.Debugf("%v %v ; received pkt\n", t.logprefix, rxpkt)
		if t.putch(t.rxch, rxpkt) == false {
			break
		}
	}
	log.Infof("%v doRx() ... stopped\n", t.logprefix)
}

func (t *Transport) unframepkt(
	conn Transporter,
	pad, packet []byte,
	tagouts map[uint64][]byte) (rxpkt rxpacket, err error) {

	var n, m int
	if n, err = io.ReadFull(conn, pad); err == io.EOF {
		log.Infof("%v doRx() received EOF\n", t.logprefix)
		return
	} else if err != nil && isConnClosed(err) {
		log.Infof("%v doRx() Closed connection\n", t.logprefix)
		atomic.AddUint64(&t.n_dropped, uint64(n))
		return
	} else if err != nil || n != 9 {
		log.Errorf("%v reading prefix: %v,%v\n", t.logprefix, n, err)
		atomic.AddUint64(&t.n_dropped, uint64(n))
		return
	} else if pad[0] != 0xd9 || pad[1] != 0xd9 || pad[2] != 0xf7 { // prefix
		err = fmt.Errorf("%v wrong prefix %v", t.logprefix, hexstring(pad))
		atomic.AddUint64(&t.n_dropped, uint64(n))
		log.Errorf("%v\n", err)
		return
	}
	//TODO: Issue #2, remove or prevent value escape to heap
	//log.Debugf("%v doRx() io.ReadFull() first %v\n", t.logprefix, pad)
	// check cbor-prefix
	n = 3
	post, request := pad[n] == 0xc6, pad[n] == 0x81
	start, stream, finish := pad[n] == 0x9f, pad[n] == 0xc7, pad[n] == 0xc8
	n += 1

	ln, m := cborItemLength(pad[n:])
	n += m

	// read the full packet
	n = copy(packet, pad[n:])
	if m, err = io.ReadFull(conn, packet[n:ln]); err == io.EOF {
		log.Infof("%v doRx() received EOF\n", t.logprefix)
		return
	} else if err != nil && isConnClosed(err) {
		log.Infof("%v doRx() Closed connection", t.logprefix)
		atomic.AddUint64(&t.n_dropped, uint64(m))
		return
	} else if err != nil || m != (int(ln)-n) {
		log.Errorf("%v reading packet %v,%v:%v\n", t.logprefix, ln, n, err)
		atomic.AddUint64(&t.n_dropped, uint64(m))
		return
	}
	atomic.AddUint64(&t.n_rxbyte, uint64(9+m))
	//TODO: Issue #2, remove or prevent value escape to heap
	//log.Debugf("%v doRx() io.ReadFull() second %v\n", t.logprefix, packet[:ln])

	// first tag is opaque
	var payload []byte
	rxpkt.opaque, payload = readtp(packet[:ln])
	rxpkt.post, rxpkt.request = post, request
	rxpkt.start, rxpkt.strmsg, rxpkt.finish = start, stream, finish
	if rxpkt.finish /*rxpkt.payload[0] == 0xff*/ { // end-of-stream
		return
	}
	// tags
	var tag uint64
	tag, payload = readtp(payload)
	for tag != tagMsg && len(payload) > 0 {
		n = t.tagdec[tag](payload, tagouts[tag])
		tag, payload = readtp(tagouts[tag][:n])
	}
	if finish == false {
		rxpkt.msg = t.unmessage(rxpkt.opaque, payload)
	}
	return
}

func (t *Transport) unmessage(opaque uint64, msgdata []byte) (bmsg BinMessage) {
	msglen := len(msgdata)
	if msglen > 0 {
		if msgdata[0] != 0xbf {
			log.Warnf("%v ##%d invalid hdr-data\n", t.logprefix, opaque)
			return
		}
	} else {
		log.Warnf("%v ##%d insufficient message length\n", t.logprefix, opaque)
		return
	}
	n := 1
	var v int
	var id int64
	var data []byte
	for (n < msglen-1) && msgdata[n] != 0xff {
		tag, k := cborItemLength(msgdata[n:])
		n += k
		switch tag {
		case tagId:
			id, v = cborItemLength(msgdata[n:])
			n += v
		case tagData:
			ln, m := cborItemLength(msgdata[n:])
			n += m
			data = msgdata[n : n+int(ln)]
			n += int(ln)
		default:
			log.Warnf("%v unknown tag in header %v,%v\n", t.logprefix, n, tag)
		}
	}
	if n >= msglen {
		log.Warnf("%v ##%d insufficient data length\n", t.logprefix, opaque)
		return
	}
	n += 1 // skip the breakstop (0xff)

	// check whether id, data is present
	if id == 0 || data == nil {
		log.Errorf("%v ##%v rx invalid message packet\n", t.logprefix, opaque)
		return
	}
	select {
	case bmsg.Data = <-t.p_data:
	default:
		bmsg.Data = make([]byte, t.buffersize)
	}
	bmsg.ID, bmsg.Data = uint64(id), bmsg.Data[:len(data)]
	copy(bmsg.Data, data)
	return
}

func (t *Transport) putch(ch chan rxpacket, val rxpacket) bool {
	select {
	case ch <- val:
		return true
	case <-t.killch:
		return false
	}
	return false
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

func readtp(payload []byte) (uint64, []byte) {
	tag, n := cborItemLength(payload)
	if tag == tagMsg {
		return uint64(tag), payload[n:]
	} else if payload[n] == 0xff {
		return uint64(tag), payload[n:]
	}
	ln, m := cborItemLength(payload[n:])
	n += m
	return uint64(tag), payload[n : n+int(ln)]
}

func (r *rxpacket) String() string {
	return fmt.Sprintf("##%d %v %v %v", r.opaque, r.request, r.start, r.finish)
}

func isConnClosed(err error) bool {
	e, ok := err.(*net.OpError)
	if ok && (e.Op == "close" || e.Op == "shutdown") {
		return true
	} else if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}
