package gofast

import "io"
import "fmt"
import "strings"
import "net"
import "sync/atomic"
import "runtime/debug"

func (t *Transport) doRx() {
	defer func() {
		if r := recover(); r != nil {
			errorf("doRx() panic: %v\n", r)
			errorf("\n%s", getStackTrace(2, debug.Stack()))
		}
		go t.Close()
	}()

	infof("%v doRx() started ...\n", t.logprefix)
	pad := make([]byte, 9)
	packet := make([]byte, t.buffersize)
	tagouts := make(map[uint64][]byte, t.buffersize)
	for _, factory := range tagFactory {
		tag, _, _ := factory(t, t.settings)
		tagouts[tag] = make([]byte, t.buffersize)
	}

	for {
		rxpkt, err := t.unframepkt(t.conn, pad, packet, tagouts)
		if err != nil {
			break
		}
		//TODO: Issue #2, remove or prevent value escape to heap
		//debugf("%v %v ; received pkt\n", t.logprefix, rxpkt)
		if t.putch(t.rxch, rxpkt) == false {
			break
		}
	}
	infof("%v doRx() ... stopped\n", t.logprefix)
}

func (t *Transport) unframepkt(
	conn Transporter,
	pad, packet []byte,
	tagouts map[uint64][]byte) (rxpkt rxpacket, err error) {

	var n, m int
	if n, err = io.ReadFull(conn, pad); err == io.EOF {
		infof("%v doRx() received EOF\n", t.logprefix)
		return
	} else if err != nil && isConnClosed(err) {
		infof("%v doRx() Closed connection\n", t.logprefix)
		atomic.AddUint64(&t.nDropped, uint64(n))
		return
	} else if err != nil || n != 9 {
		errorf("%v reading prefix: %v,%v\n", t.logprefix, n, err)
		atomic.AddUint64(&t.nDropped, uint64(n))
		return
	} else if pad[0] != 0xd9 || pad[1] != 0xd9 || pad[2] != 0xf7 { // prefix
		err = fmt.Errorf("%v wrong prefix %v", t.logprefix, hexstring(pad))
		atomic.AddUint64(&t.nDropped, uint64(n))
		errorf("%v\n", err)
		return
	}
	//TODO: Issue #2, remove or prevent value escape to heap
	//debugf("%v doRx() io.ReadFull() first %v\n", t.logprefix, pad)
	// check cbor-prefix
	n = 3
	post, request := pad[n] == 0xc6, pad[n] == 0x81
	start, stream, finish := pad[n] == 0x9f, pad[n] == 0xc7, pad[n] == 0xc8
	n++

	ln, m := cborItemLength(pad[n:])
	n += m
	if finish { // railing 0xff
		ln++
	}

	// read the full packet
	n = copy(packet, pad[n:])
	if m, err = io.ReadFull(conn, packet[n:ln]); err == io.EOF {
		infof("%v doRx() received EOF\n", t.logprefix)
		return
	} else if err != nil && isConnClosed(err) {
		infof("%v doRx() Closed connection\n", t.logprefix)
		atomic.AddUint64(&t.nDropped, uint64(m))
		return
	} else if err != nil || m != (int(ln)-n) {
		errorf("%v reading packet %v,%v:%v\n", t.logprefix, ln, n, err)
		atomic.AddUint64(&t.nDropped, uint64(m))
		return
	}
	atomic.AddUint64(&t.nRxbyte, uint64(9+m))
	//TODO: Issue #2, remove or prevent value escape to heap
	//debugf("%v doRx() io.ReadFull() second %v\n", t.logprefix, packet[:ln])

	// first tag is opaque
	var payload []byte
	rxpkt.opaque, payload = readtp(packet[:ln])
	rxpkt.post, rxpkt.request = post, request
	rxpkt.start, rxpkt.strmsg, rxpkt.finish = start, stream, finish
	if rxpkt.finish { // end-of-stream
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
			warnf("%v ##%d invalid hdr-data\n", t.logprefix, opaque)
			return
		}
	} else {
		warnf("%v ##%d insufficient message length\n", t.logprefix, opaque)
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
		case tagID:
			id, v = cborItemLength(msgdata[n:])
			n += v
		case tagData:
			ln, m := cborItemLength(msgdata[n:])
			n += m
			data = msgdata[n : n+int(ln)]
			n += int(ln)
		default:
			warnf("%v unknown tag in header %v,%v\n", t.logprefix, n, tag)
		}
	}
	if n >= msglen {
		warnf("%v ##%d insufficient data length\n", t.logprefix, opaque)
		return
	}
	n++ // skip the breakstop (0xff)

	// check whether id, data is present
	if id == 0 || data == nil {
		errorf("%v ##%v rx invalid message packet\n", t.logprefix, opaque)
		return
	}
	bmsg.ID = uint64(id)
	bmsg.Data = t.getdata(len(data))
	copy(bmsg.Data, data)
	return
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

func isConnClosed(err error) bool {
	e, ok := err.(*net.OpError)
	if ok && (e.Op == "close" || e.Op == "shutdown") {
		return true
	} else if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}
