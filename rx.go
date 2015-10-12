package gofast

import "sync"
import "io"
import "fmt"

var rxpool = sync.Pool{New: func() interface{} { return &rxpacket{} }}

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
			fmsg := "%v stream ##%d closed ...\n"
			log.Debugf(fmsg, t.logprefix, stream.opaque)
			delete(livestreams, stream.opaque)

		} else {
			fmsg := "%v ##%d stream started ...\n"
			log.Verbosef(fmsg, t.logprefix, stream.opaque)
			livestreams[stream.opaque] = stream
		}
	}

	handlepkt := func(rxpkt *rxpacket) {
		stream, streamok := livestreams[rxpkt.opaque]
		defer t.pktpool.Put(rxpkt.packet)
		defer rxpool.Put(rxpkt)

		if rxpkt.finish {
			fmsg := "%v stream ##%d closed by remote ...\n"
			log.Debugf(fmsg, t.logprefix, stream.opaque)
			t.putstream(rxpkt.opaque, stream, false /*tellrx*/)
			delete(livestreams, rxpkt.opaque)
			return

		}
		msg := t.unmessage(rxpkt.opaque, rxpkt.payload)
		if msg == nil {
			return
		}
		log.Debugf("%v received msg %v\n", t.logprefix, msg)
		if rxpkt.request || rxpkt.start {
			if !streamok {
				stream = t.newstream(rxpkt.opaque, true)
				livestreams[stream.opaque] = stream
				stream.Rxch = t.handlers[msg.Id()](stream, msg)
				return
			}
			fmsg := "%v ##%d stream already active ...\n"
			log.Warnf(fmsg, t.logprefix, rxpkt.opaque)

		} else if streamok == false {
			fmsg := "%v ##%d stream unknown ...\n"
			log.Debugf(fmsg, t.logprefix, rxpkt.opaque)
		}
		t.putmsg(stream.Rxch, msg)
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
		rxpkt := t.unframepkt(t.conn)
		log.Debugf("%v received packet %v\n", t.logprefix, rxpkt)
		if t.putch(t.rxch, rxpkt) == false {
			break
		}
	}
	log.Infof("%v doRx() ... stopped\n", t.logprefix)
}

func (t *Transport) unframepkt(conn Transporter) (rxpkt *rxpacket) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("%v malformed packet: %v\n", t.logprefix, r)
			if rxpkt != nil {
				t.pktpool.Put(rxpkt.packet)
				rxpool.Put(rxpkt)
			}
		}
	}()

	var pad [9]byte
	if n, err := io.ReadFull(conn, pad[:9]); err == io.EOF {
		log.Infof("%v doRx() received EOF\n", t.logprefix)
		return
	} else if err != nil || n != 9 {
		log.Errorf("%v reading prefix: %v,%v\n", t.logprefix, n, err)
		return
	}
	//fmt.Println("doRx() io.ReadFull() first", pad)
	// check cbor-prefix
	if pad[0] != 0xd9 || pad[1] != 0xd9 || pad[2] != 0xf7 {
		log.Errorf("%v wrong prefix %v\n", t.logprefix, pad)
		return
	}
	n := 3
	request, start := pad[n] == 0x91, pad[n] == 0x9f
	if request || start {
		n += 1
	}
	ln, m := cborItemLength(pad[n:])
	n += m
	// read the full packet
	packet := t.pktpool.Get().([]byte)
	defer func() { t.pktpool.Put(packet) }()
	n = copy(packet, pad[n:9])
	if m, err := io.ReadFull(conn, packet[n:ln]); err == io.EOF {
		log.Infof("%v doRx() received EOF\n", t.logprefix)
		return
	} else if err != nil || m != (ln-n) {
		log.Errorf("%v reading packet %v,%v,%v\n", t.logprefix, ln, n, err)
		return
	}
	//fmt.Println("doRx() io.ReadFull() second")
	rxpkt = rxpool.Get().(*rxpacket)
	rxpkt.packet = t.pktpool.Get().([]byte)
	// opaque
	opaque, payload := readtp(packet)
	rxpkt.request, rxpkt.start, rxpkt.opaque = request, start, opaque
	if payload[0] == 0xff { // end-of-stream
		rxpkt.finish = true
		return
	}
	// tags
	var tag uint64
	tag, rxpkt.payload = readtp(payload)
	rxpkt.packet, packet = packet, rxpkt.packet // swap
	for tag != tagMsg && len(payload) > 0 {
		n = t.tagdec[tag](rxpkt.payload, packet)
		tag, rxpkt.payload = readtp(packet)
		rxpkt.packet, packet = packet, rxpkt.packet // swap
	}
	return rxpkt
}

func (t *Transport) unmessage(opaque uint64, msgdata []byte) Message {
	if msgdata[0] != 0xbf {
		log.Warnf("%v ##%d invalid hdr-data\n", t.logprefix, opaque)
		return nil
	}
	n := 1
	var v, id int
	var data []byte
	for msgdata[n] != 0xff {
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
	n += 1 // skip the breakstop (0xff)

	// check whether id, data is present
	if id == 0 || data == nil {
		log.Errorf("%v %v rx invalid message packet\n", opaque, t.logprefix)
		return nil
	}
	obj := t.msgpools[uint64(id)].Get()
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
