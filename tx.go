package gofast

import "sync"
import "fmt"

// | 0xd9f7 | packet |
func (t *Transport) post(msg Message, stream *Stream, out []byte) (n int) {
	n = tag2cbor(tagCborPrefix, out)      // prefix
	n += t.framepkt(stream, msg, out[n:]) // packet
	return n
}

// | 0xd9f7 | 0x91 | packet |
func (t *Transport) request(msg Message, stream *Stream, out []byte) (n int) {
	n = tag2cbor(tagCborPrefix, out) // prefix
	out[n] = 0x91                    // 0x91
	n += 1
	n += t.framepkt(stream, msg, out[n:]) // packet
	return n
}

// | 0xd9f7 | 0x91 | packet |
func (t *Transport) response(msg Message, stream *Stream, out []byte) (n int) {
	n = tag2cbor(tagCborPrefix, out) // prefix
	out[n] = 0x91                    // 0x91
	n += 1
	n += t.framepkt(stream, msg, out[n:]) // packet
	return n
}

// | 0xd9f7 | 0x9f | packet1 |
func (t *Transport) start(msg Message, stream *Stream, out []byte) (n int) {
	n = tag2cbor(tagCborPrefix, out)      // prefix
	n += arrayStart(out[n:])              // 0x9f
	n += t.framepkt(stream, msg, out[n:]) // packet
	return n
}

// | 0xd9f7 | packet2 |
func (t *Transport) stream(stream *Stream, msg Message, out []byte) (n int) {
	n = tag2cbor(tagCborPrefix, out)      // prefix
	n += t.framepkt(stream, msg, out[n:]) // packet
	return n
}

// | 0xd9f7 | packetN | 0xff |
func (t *Transport) finish(stream *Stream, out []byte) (n int) {
	var scratch [16]byte
	n = tag2cbor(tagCborPrefix, out)         // prefix
	m := tag2cbor(stream.opaque, scratch[:]) // tag-opaque
	scratch[m] = 0xff                        // 0xff (payload)
	m += 1
	n += value2cbor(scratch[:m], out[n:]) // packet
	return n
}

func (t *Transport) framepkt(stream *Stream, msg Message, ping []byte) (n int) {
	// compose message
	data := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(data)
	// create another buffer and rotate with `ping` buffer
	// and roll up the tags
	pong := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(pong)

	// tagMsg
	n = tag2cbor(tagMsg, ping) // tagMsg
	n += mapStart(ping[n:])
	n += value2cbor(tagId, ping[n:]) // hdr-tagId
	n += value2cbor(msg.Id(), ping[n:])
	p := msg.Encode(data)
	n += value2cbor(tagData, ping[n:]) // hdr-tagId
	n += value2cbor(data[:p], ping[n:])
	n += breakStop(ping[n:])

	var m int
	for tag, fn := range t.tagenc { // roll up tags
		if m = fn(ping[:n], pong); n == 0 { // skip tag
			continue
		}
		n = tag2cbor(tag, ping)
		n += value2cbor(pong[:m], ping[n:])
	}

	m = tag2cbor(stream.opaque, pong) // finally roll up opaque
	m += value2cbor(ping[:n], pong[m:])
	return value2cbor(pong[:m], ping) // packet encoded as CBOR byte array
}

var txpool = sync.Pool{New: func() interface{} { return &txproto{} }}

type txproto struct {
	packet []byte // request
	flush  bool
	n      int // response
	err    error
	respch chan *txproto
}

func (t *Transport) tx(packet []byte, flush bool) (err error) {
	arg := txpool.Get().(*txproto)
	defer txpool.Put(arg)

	log.Debugf("%v tx packet %v\n", t.logprefix, packet)
	arg.packet, arg.flush, arg.respch = packet, flush, make(chan *txproto, 1)
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

func (t *Transport) doTx() {
	batchsize := t.config["batchsize"].(int)
	buffersize := t.config["buffersize"].(int)
	batch, buffersize, buffercap := make([]*txproto, 0, batchsize), 0, buffersize
	drainbuffers := func() {
		for _, arg := range batch {
			arg.n, arg.err = 0, nil
			if len(arg.packet) > 0 { // don't send it empty
				n, err := t.conn.Write(arg.packet)
				arg.n, arg.err = n, err
			}
			arg.respch <- arg
		}
		log.Debugf("%v drained %v packets\n", t.logprefix, len(batch))
		// reset the batch
		buffersize = 0
		batch = batch[:0]
	}

	fmsg := "%v doTx(batch:%v, buffer:%v) started ...\n"
	log.Infof(fmsg, t.logprefix, batchsize, buffercap)
loop:
	for {
		select {
		case arg := <-t.txch:
			batch = append(batch, arg)
			buffersize += len(arg.packet)
			if arg.flush || len(batch) >= batchsize || buffersize > buffercap {
				drainbuffers()
			}

		case <-t.killch:
			break loop
		}
	}
	log.Infof("%v doTx() ... stopped\n", t.logprefix)
}
