package gofast

// | 0xd9f7 | packet |
func (t *Transport) post(msg Message) error {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)

	stream := t.getstream(nil)
	defer t.putstream(stream)

	n := tag2cbor(tagCborPrefix, out) // prefix
	m := frame(stream, msg, out[n:])  // packet
	n += m
	if m > 0 {
		return t.tx(out[:n], false)
	}
	return fmt.Errorf("empty packet")
}

// | 0xd9f7 | 0x9f | packet | 0xff |
func (t *Transport) request(msg Message) (respmsg Message, err error) {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)

	stream = t.getstream(make(chan Message, 1))
	defer func() {
		if err != nil {
			t.putstream(stream)
		}
	}()

	n := tag2cbor(tagCborPrefix, out) // prefix
	n += arrayStart(out[n:])          // 0x9f
	m := frame(stream, msg, out[n:])  // packet
	if m > 0 {
		n += m
		n += breakStop(out[n:]) // 0xff
		if err = t.tx(out[:n], false); err != nil {
			t.putstream(stream)
			return nil, err
		}
		respmsg <- stream.Rxch
		return respmsg, nil
	}
	err = fmt.Errorf("empty packet")
	return nil, err
}

// | 0xd9f7 | 0x9f | packet1 |
func (t *Transport) start(
	msg Message, rxch chan Message) (stream *Stream, err error) {

	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)

	stream = t.getstream(rxch)
	defer func() {
		if err != nil {
			t.putstream(stream)
		}
	}()

	n := tag2cbor(tagCborPrefix, out) // prefix
	n += arrayStart(out[n:])          // 0x9f
	m := frame(stream, msg, out[n:])  // packet
	if m > 0 {
		n += m
		if err = t.tx(out[:n], false); err != nil {
			t.putstream(stream)
			return nil, err
		}
		return
	}
	err = fmt.Errorf("empty packet")
	return nil, err
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

	n := tag2cbor(tagCborPrefix, out) // prefix
	m := frame(stream, msg, out[n:])  // packet
	if m > 0 {
		n += m
		if err = t.tx(out[:n], false); err != nil {
			t.putstream(stream)
			return err
		}
		return
	}
	err = fmt.Errorf("empty packet")
	return err
}

// | 0xd9f7 | packetN | 0xff |
func (t *Transport) finish(stream *Stream) error {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)

	n := tag2cbor(tagCborPrefix, out) // prefix
	n += breakStop(out[n:])           // 0xff
	err := t.tx(out[:n], false)
	t.putstream(stream)
	return err
}

func (t *Transport) frame(stream *Stream, msg Message, out []byte) (n int) {
	// compose message
	data := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(data)
	if p := msg.Encode(data); p == 0 {
		return 0
	} else {
		stream.blueprint[tagId] = msg.Id()
		stream.blueprint[tagData] = data[:p]
	}
	n += mapStart(out[n:])
	for key, value := range stream.blueprint {
		n += value2cbor(key, out[n:])
		n += value2cbor(value, out[n:])
	}
	n += breakStop(out[n:])

	// create another buffer and rotate with `out` buffer and roll up the tags
	packet := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(packet)

	m := tag2cbor(tagMsg, packet) // roll up tagMsg
	m += value2cbor(out[:n], packet[m:])
	for _, tag := range t.tags { // roll up tags
		if doenc, ok := t.tagok[tag]; ok && doenc { // if requested by peer
			if fn, ok := t.tagenc[tag]; ok {
				n = fn(packet[:m], out)
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

var txpool = sync.Pool{New: func() interface{} { return &txreq{} }}

type txproto struct {
	packet []byte // request
	flush  bool
	count  int // response
	err    error
	respch chan *txproto
}

func (t *Transport) tx(packet []byte, flush bool) (n int, err error) {
	arg := txpool.Get().(*txproto)
	defer txpool.Put(arg)

	arg.packet, arg.flush, arg.respch = packet, flush, make(chan *txproto, 1)
	t.txch <- arg
	resp := <-arg.respch
	n, err = resp.count, resp.err
	return n, err
}

func (t *Transport) doTx() {
	batchlen := t.config["batchlen"].(int)
	buflen := t.config["buflen"].(int)
	batch, buflen, bufcap = make([]*txproto, 0, batchlen), 0, buflen
	drainbuffers := func() {
		for _, arg := range batch {
			arg.count, arg.err = 0, nil
			if len(arg.packet) > 0 { // don't send it empty
				n, err := t.conn.Write(arg.packet)
				arg.count, arg.err = n, err
			}
			arg.respch <- arg
		}
		// reset the batch
		buflen = 0
		batch = batch[:0]
	}

	for {
		arg, ok := <-t.txch
		if ok == false {
			break
		}
		batch = append(batch, arg)
		buflen += len(arg.packet)
		if arg.flush || len(batch) >= batchlen || buflen > bufcap {
			drainbuffers()
		}
	}
}
