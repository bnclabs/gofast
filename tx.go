package gofast

import "fmt"
import "sync/atomic"

// | 0xd9f7 | packet |
func (t *Transport) post(msg Message, stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txpost, 1)
	n = tag2cbor(tagCborPrefix, out)      // prefix
	n += t.framepkt(msg, stream, out[n:]) // packet
	return n
}

// | 0xd9f7 | 0x91 | packet |
func (t *Transport) request(msg Message, stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txreq, 1)
	n = tag2cbor(tagCborPrefix, out) // prefix
	out[n] = 0x91                    // 0x91
	n += 1
	n += t.framepkt(msg, stream, out[n:]) // packet
	return n
}

// | 0xd9f7 | 0x91 | packet |
func (t *Transport) response(msg Message, stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txresp, 1)
	n = tag2cbor(tagCborPrefix, out) // prefix
	out[n] = 0x91                    // 0x91
	n += 1
	n += t.framepkt(msg, stream, out[n:]) // packet
	return n
}

// | 0xd9f7 | 0x9f | packet1 |
func (t *Transport) start(msg Message, stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txstart, 1)
	n = tag2cbor(tagCborPrefix, out)      // prefix
	n += arrayStart(out[n:])              // 0x9f
	n += t.framepkt(msg, stream, out[n:]) // packet
	return n
}

// | 0xd9f7 | packet2 |
func (t *Transport) stream(msg Message, stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txstream, 1)
	n = tag2cbor(tagCborPrefix, out)      // prefix
	n += t.framepkt(msg, stream, out[n:]) // packet
	return n
}

// | 0xd9f7 | packetN | 0xff |
func (t *Transport) finish(stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txfin, 1)
	var scratch [16]byte
	n = tag2cbor(tagCborPrefix, out)         // prefix
	m := tag2cbor(stream.opaque, scratch[:]) // tag-opaque
	scratch[m] = 0xff                        // 0xff (payload)
	m += 1
	n += valbytes2cbor(scratch[:m], out[n:]) // packet
	return n
}

func (t *Transport) framepkt(msg Message, stream *Stream, ping []byte) (n int) {
	// data
	x := t.p_txdata.Get()
	defer t.p_txdata.Put(x)
	// create another buffer and rotate with `ping` buffer
	// and roll up the tags
	y := t.p_txdata.Get()
	defer t.p_txdata.Put(y)

	data, pong := x.([]byte), y.([]byte)

	// tagMsg
	n = tag2cbor(tagMsg, ping) // tagMsg
	n += mapStart(ping[n:])
	n += valuint642cbor(tagId, ping[n:])    // hdr-tagId
	n += valuint642cbor(msg.Id(), ping[n:]) // value
	n += valuint642cbor(tagData, ping[n:])  // hdr-tagData
	m := msg.Encode(data)                   // value
	n += valbytes2cbor(data[:m], ping[n:])
	n += breakStop(ping[n:])

	for tag, fn := range t.tagenc { // roll up tags
		if m = fn(ping[:n], pong); m == 0 { // skip tag
			continue
		}
		n = tag2cbor(tag, ping)
		n += valbytes2cbor(pong[:m], ping[n:])
	}

	m = tag2cbor(stream.opaque, pong) // finally roll up opaque
	m += valbytes2cbor(ping[:n], pong[m:])
	n = valbytes2cbor(pong[:m], ping) // packet encoded as CBOR byte array
	return n
}

type txproto struct {
	packet []byte // request
	flush  bool
	async  bool
	n      int // response
	err    error
	respch chan *txproto
}

func (t *Transport) tx(packet []byte, flush bool) (err error) {
	arg := fromtxpool(false /*async*/, t.p_txcmd)
	defer func() { arg.packet = nil; t.p_txcmd.Put(arg) }()

	arg.packet, arg.flush, arg.async = packet, flush, false
	arg.respch = make(chan *txproto, 1)
	select {
	case t.txch <- arg:
		select {
		case resp := <-arg.respch:
			n, err := resp.n, resp.err
			if err == nil && n != len(packet) {
				return fmt.Errorf("partial write")
			}
			return err // success or error
		case <-t.killch:
			return fmt.Errorf("transport closed")
		}

	case <-t.killch:
		return fmt.Errorf("transport closed")
	}
	return nil
}

func (t *Transport) txasync(out []byte, flush bool) (err error) {
	arg := fromtxpool(true /*async*/, t.p_txacmd)
	if arg.packet == nil {
		arg.packet = t.p_txdata.Get().([]byte)
	}
	arg.packet = arg.packet[:cap(arg.packet)]

	n := copy(arg.packet, out)
	arg.packet = arg.packet[:n]

	arg.flush, arg.async = flush, true
	select {
	case t.txch <- arg:
	case <-t.killch:
		return fmt.Errorf("transport closed")
	}
	return nil
}

func (t *Transport) doTx() {
	batchsize := t.config["batchsize"].(int)
	buffersize := t.config["buffersize"].(int)
	batch := make([]*txproto, 0, 64)
	tcpwrite_buf := make([]byte, batchsize*buffersize)
	drainbuffers := func() {
		atomic.AddUint64(&t.n_flushes, 1)
		var err error
		m, n := 0, 0
		// consolidate.
		for _, arg := range batch {
			if len(arg.packet) > 0 {
				n += copy(tcpwrite_buf[n:], arg.packet)
				atomic.AddUint64(&t.n_tx, 1)
			}
		}
		// send.
		if n > 0 {
			fmsg := "%v doTx() socket write %v:%v\n"
			log.Debugf(fmsg, t.logprefix, n, tcpwrite_buf[:n])
			m, err = t.conn.Write(tcpwrite_buf[:n])
			if m != n {
				err = fmt.Errorf("wrote only %d, expected %d", m, n)
			}
		}
		atomic.AddUint64(&t.n_txbyte, uint64(m))
		// unblock the callers.
		for _, arg := range batch {
			arg.n, arg.err = len(arg.packet), err
			if arg.async {
				t.p_txacmd.Put(arg)
			} else {
				arg.respch <- arg
			}
		}
		log.Debugf("%v drained %v packets\n", t.logprefix, len(batch))
		batch = batch[:0] // reset the batch
	}

	log.Infof("%v doTx(batch:%v) started ...\n", t.logprefix, batchsize)
loop:
	for {
		select {
		case arg := <-t.txch:
			batch = append(batch, arg)
			if arg.flush || len(batch) >= batchsize {
				drainbuffers()
			}

		case <-t.killch:
			break loop
		}
	}
	log.Infof("%v doTx() ... stopped\n", t.logprefix)
}
