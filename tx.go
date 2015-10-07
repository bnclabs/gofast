package gofast

import "sync"
import "fmt"
import "errors"

// | 0xd9f7 | packet |
func (t *Transport) post(msg Message) error {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)

	stream := t.getstream(nil)
	defer t.putstream(stream)

	n := tag2cbor(tagCborPrefix, out)  // prefix
	n += t.frame(stream, msg, out[n:]) // packet
	return t.tx(out[:n], false)
}

// | 0xd9f7 | 0x9f | packet | 0xff |
func (t *Transport) request(msg Message) (respmsg Message, err error) {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)

	stream := t.getstream(make(chan Message, 1))
	defer func() {
		if err != nil {
			t.putstream(stream)
		}
	}()

	n := tag2cbor(tagCborPrefix, out)  // prefix
	n += arrayStart(out[n:])           // 0x9f
	n += t.frame(stream, msg, out[n:]) // packet
	n += breakStop(out[n:])            // 0xff
	if err = t.tx(out[:n], false); err != nil {
		t.putstream(stream)
		return nil, err
	}
	respmsg, ok := <-stream.Rxch
	if ok {
		return respmsg, nil
	}
	return nil, errors.New("stream closed")
}

// | 0xd9f7 | 0x9f | packet1 |
func (t *Transport) start(msg Message, ch chan Message) (stream *Stream, err error) {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)

	stream = t.getstream(ch)
	defer func() {
		if err != nil {
			t.putstream(stream)
		}
	}()

	n := tag2cbor(tagCborPrefix, out)  // prefix
	n += arrayStart(out[n:])           // 0x9f
	n += t.frame(stream, msg, out[n:]) // packet
	if err = t.tx(out[:n], false); err != nil {
		t.putstream(stream)
		return nil, err
	}
	return stream, nil
}

// | 0xd9f7 | packet2 |
func (t *Transport) stream(stream *Stream, msg Message) (err error) {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)

	defer func() {
		if err != nil {
			t.putstream(stream)
		}
	}()

	n := tag2cbor(tagCborPrefix, out)  // prefix
	n += t.frame(stream, msg, out[n:]) // packet
	if err = t.tx(out[:n], false); err != nil {
		t.putstream(stream)
		return err
	}
	return nil
}

// | 0xd9f7 | packetN | 0xff |
func (t *Transport) finish(stream *Stream) error {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)

	n := tag2cbor(tagCborPrefix, out)  // prefix
	n += t.frame(stream, nil, out[n:]) // packet
	n += breakStop(out[n:])            // 0xff
	err := t.tx(out[:n], false)
	t.putstream(stream)
	return err
}

func (t *Transport) frame(stream *Stream, msg Message, out []byte) (n int) {
	// compose message
	data := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(data)
	// create another buffer and rotate with `out` buffer
	// and roll up the tags
	packet := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(packet)

	p, m := 0, 0
	if msg != nil {
		p = msg.Encode(data)
	}
	stream.blueprint[tagId] = msg.Id()
	stream.blueprint[tagData] = data[:p]
	n += mapStart(out[n:])
	for key, value := range stream.blueprint {
		n += value2cbor(key, out[n:])
		n += value2cbor(value, out[n:])
	}
	n += breakStop(out[n:])

	m = tag2cbor(tagMsg, packet) // roll up tagMsg
	m += value2cbor(out[:n], packet[m:])
	for tag := range t.tags { // roll up tags
		if doenc, ok := t.tagok[tag]; ok && doenc { // if requested by peer
			if fn, ok := t.tagenc[tag]; ok {
				if n = fn(packet[:m], out); n == 0 { // skip tag
					continue
				}
				m = tag2cbor(tag, packet)
				m += value2cbor(out[:n], packet[m:])
			}
		}
	}

	// finally roll up tagOpaqueStart-tagOpaqueEnd
	n = tag2cbor(stream.opaque, out)
	n += value2cbor(packet[:m], out[n:])
	return n
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
			return errors.New("transport closed")
		}
	case <-t.killch:
		return errors.New("transport closed")
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
		// reset the batch
		buffersize = 0
		batch = batch[:0]
	}

	fmsg := "%v doTx(batch:%v, buffer:%v) started ...\n"
	log.Infof(fmsg, t.logprefix, batchsize, buffercap)
loop:
	for {
		arg, ok := <-t.txch
		if ok == false {
			break loop
		}
		batch = append(batch, arg)
		buffersize += len(arg.packet)
		if arg.flush || len(batch) >= batchsize || buffersize > buffercap {
			drainbuffers()
		}
	}
	log.Infof("%v doTx() ... stopped\n", t.logprefix)
}
