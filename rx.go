package gofast

import "sync/atomic"
import "io"
import "fmt"

type rxpacket struct {
	packet  []byte
	payload []byte
	opaque  uint64
	request bool
	start   bool
	finish  bool
}

func (t *Transport) syncRx() {
	chansize := t.config["chansize"].(int)
	livestreams := make(map[uint64]*Stream)

	streamupdate := func(stream *Stream) {
		_, ok := livestreams[stream.opaque]
		if stream.Rxch == nil && ok {
			log.Debugf("%v ##%d stream closed ...\n", t.logprefix, stream.opaque)
			delete(livestreams, stream.opaque)
			if stream.remote == false {
				t.strmpool <- stream
			}
			return
		}
		log.Verbosef("%v ##%d stream started ...\n", t.logprefix, stream.opaque)
		livestreams[stream.opaque] = stream
	}

	handlepkt := func(rxpkt *rxpacket) {
		defer t.pktpool.Put(rxpkt.packet)
		defer t.p_rxcmd.Put(rxpkt)

		stream, streamok := livestreams[rxpkt.opaque]

		if rxpkt.finish {
			fmsg := "%v ##%d stream closed by remote ...\n"
			log.Debugf(fmsg, t.logprefix, stream.opaque)
			t.putstream(rxpkt.opaque, stream, false /*tellrx*/)
			delete(livestreams, rxpkt.opaque)
			atomic.AddUint64(&t.n_rxfin, 1)
			return
		}

		msg := t.unmessage(rxpkt.opaque, rxpkt.payload)
		if msg == nil {
			return
		}
		fmsg := "%v received msg %#v streamok:%v\n"
		log.Debugf(fmsg, t.logprefix, msg, streamok)
		if streamok == false { // post, request, stream-start
			if rxpkt.request {
				func() {
					stream = t.newstream(rxpkt.opaque, true /*remote*/)
					defer rxstrmpool.Put(stream)
					t.handlers[msg.Id()](stream, msg)
					atomic.AddUint64(&t.n_rxreq, 1)
				}()
			} else if rxpkt.start {
				stream = t.newstream(rxpkt.opaque, true /*remote*/)
				stream.Rxch = t.handlers[msg.Id()](stream, msg)
				livestreams[stream.opaque] = stream
				atomic.AddUint64(&t.n_rxstart, 1)
			} else {
				t.handlers[msg.Id()](nil /*stream*/, msg)
				atomic.AddUint64(&t.n_rxpost, 1)
			}
			return
		}
		// response, stream, finish
		t.putmsg(stream.Rxch, msg)
		if rxpkt.request {
			atomic.AddUint64(&t.n_rxresp, 1)
		} else if rxpkt.finish {
			atomic.AddUint64(&t.n_rxfin, 1)
		} else {
			atomic.AddUint64(&t.n_rxstream, 1)
		}
	}

	go t.doRx()

	fmsg := "%v syncRx(chansize:%v) started ...\n"
	log.Infof(fmsg, t.logprefix, chansize)
loop:
	for {
		select {
		case arg := <-t.rxch:
			switch val := arg.(type) {
			case *Stream:
				streamupdate(val)
			case *rxpacket:
				if val != nil {
					atomic.AddUint64(&t.n_rx, 1)
					handlepkt(val)
				}
			}
		case <-t.killch:
			break loop
		}
	}
	log.Infof("%v syncRx() ... stopped\n", t.logprefix)
}

func (t *Transport) doRx() {
	defer func() { go t.Close() }()

	log.Infof("%v doRx() started ...\n", t.logprefix)
	for {
		rxpkt, err := t.unframepkt(t.conn)
		if err != nil {
			break
		}
		if rxpkt != nil {
			log.Debugf("%v %v ; received pkt\n", t.logprefix, rxpkt)
			if t.putch(t.rxch, rxpkt) == false {
				break
			}
		}
	}
	log.Infof("%v doRx() ... stopped\n", t.logprefix)
}

func (t *Transport) unframepkt(conn Transporter) (rxpkt *rxpacket, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("%v malformed packet: %v\n", t.logprefix, r)
			if rxpkt != nil {
				t.pktpool.Put(rxpkt.packet)
				t.p_rxcmd.Put(rxpkt)
			}
			rxpkt, err = nil, fmt.Errorf("%v", r)
		}
	}()

	var n int
	var pad [9]byte
	if n, err = io.ReadFull(conn, pad[:9]); err == io.EOF {
		log.Infof("%v doRx() received EOF\n", t.logprefix)
		return
	} else if err != nil || n != 9 {
		log.Errorf("%v reading prefix: %v,%v\n", t.logprefix, n, err)
		atomic.AddUint64(&t.n_dropped, uint64(n))
		return
	} else if pad[0] != 0xd9 || pad[1] != 0xd9 || pad[2] != 0xf7 { // prefix
		err = fmt.Errorf("%v wrong prefix %v", t.logprefix, pad)
		atomic.AddUint64(&t.n_dropped, uint64(n))
		log.Errorf("%v\n", err)
		return
	}
	log.Debugf("%v doRx() io.ReadFull() first %v\n", t.logprefix, pad)
	// check cbor-prefix
	n = 3
	request, start := pad[n] == 0x91, pad[n] == 0x9f
	if request || start {
		n += 1
	}
	ln, m := cborItemLength(pad[n:])
	n += m
	// read the full packet
	packet := t.pktpool.Get().([]byte)
	n = copy(packet, pad[n:9])
	if m, err = io.ReadFull(conn, packet[n:ln]); err == io.EOF {
		log.Infof("%v doRx() received EOF\n", t.logprefix)
		return
	} else if err != nil || m != (ln-n) {
		log.Infof("%v reading packet %v,%v:%v\n", t.logprefix, ln, n, err)
		atomic.AddUint64(&t.n_dropped, uint64(m))
		return
	}
	atomic.AddUint64(&t.n_rxbyte, uint64(9+m))
	log.Debugf("%v doRx() io.ReadFull() second %v\n", t.logprefix, packet[:ln])
	rxpkt = fromrxpool(t.p_rxcmd)
	rxpkt.packet = packet
	// first tag is opaque
	rxpkt.opaque, rxpkt.payload = readtp(rxpkt.packet)
	rxpkt.request, rxpkt.start = request, start
	if rxpkt.payload[0] == 0xff { // end-of-stream
		rxpkt.finish = true
		return
	}
	// tags
	var tag uint64
	tag, rxpkt.payload = readtp(rxpkt.payload)
	for tag != tagMsg && len(rxpkt.payload) > 0 {
		func() {
			defer t.pktpool.Put(rxpkt.packet)
			packet = t.pktpool.Get().([]byte)
			n = t.tagdec[tag](rxpkt.payload, packet)
			tag, rxpkt.payload = readtp(packet[:n])
			rxpkt.packet = packet
		}()
	}
	return rxpkt, nil
}

func (t *Transport) unmessage(opaque uint64, msgdata []byte) Message {
	msglen := len(msgdata)
	if msglen > 0 {
		if msgdata[0] != 0xbf {
			log.Warnf("%v ##%d invalid hdr-data\n", t.logprefix, opaque)
			return nil
		}
	} else {
		log.Warnf("%v ##%d insufficient message length\n", t.logprefix, opaque)
		return nil
	}
	n := 1
	var v, id int
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
			data = msgdata[n : n+ln]
			n += ln
		default:
			log.Warnf("%v unknown tag in header %v,%v\n", t.logprefix, n, tag)
		}
	}
	if n >= msglen {
		log.Warnf("%v ##%d insufficient data length\n", t.logprefix, opaque)
		return nil
	}
	n += 1 // skip the breakstop (0xff)

	// check whether id, data is present
	if id == 0 || data == nil {
		log.Errorf("%v ##%v rx invalid message packet\n", t.logprefix, opaque)
		return nil
	}
	pool, ok := t.msgpools[uint64(id)]
	if !ok {
		log.Errorf("%v ##%v rx invalid message id %v\n", t.logprefix, opaque, id)
		return nil
	}
	obj := pool.Get()
	if m, ok := obj.(*Whoami); ok { // hack to make whoami aware of transport
		m.transport = t
	}
	msg := obj.(Message)
	msg.Decode(data)
	return msg
}

func (t *Transport) putch(ch chan interface{}, val interface{}) bool {
	select {
	case ch <- val:
		return true
	case <-t.killch:
		return false
	}
	return false
}

func (t *Transport) getch(ch chan interface{}) interface{} {
	select {
	case val := <-ch:
		return val
	case <-t.killch:
		return nil
	}
	return nil
}

func (t *Transport) putmsg(ch chan Message, msg Message) bool {
	select {
	case ch <- msg:
		return true
	case <-t.killch:
		return false
	}
	return false
}

func readtp(payload []byte) (uint64, []byte) {
	tag, n := cborItemLength(payload)
	if tag == tagMsg {
		return uint64(tag), payload[n:]
	}
	ln, m := cborItemLength(payload[n:])
	n += m
	return uint64(tag), payload[n : n+ln]
}

func (r *rxpacket) String() string {
	return fmt.Sprintf("##%d %v %v %v", r.opaque, r.request, r.start, r.finish)
}
