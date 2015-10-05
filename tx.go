package gofast

// | 0xd9f7 | packet |
func (t *Transport) post(msg Message, out []byte) error {
	stream := t.get(nil)

	n := tag2cbor(tagCborPrefix, out)
	n += frame(stream, msg, out[n:])
	err := t.tx(out[:n], false)
	t.put(stream)
	return err
}

// | 0xd9f7 | 0x9f | packet | 0xff |
func (t *Transport) request(msg Message, out []byte) (*Stream, error) {
	stream := t.get(make(chan Message, 1))

	n := tag2cbor(tagCborPrefix, out)
	n += arrayStart(out[n:])
	n += frame(stream, msg, out[n:])
	n += breakStop(out[n:])
	if err := t.tx(out[:n], false); err != nil {
		t.put(stream)
		return nil, err
	}
	return stream, nil
}

// | 0xd9f7 | 0x9f | packet1 |
func (t *Transport) start(msg Message, out []byte, rxch chan Message) (*Stream, error) {
	stream := t.get(rxch)

	n := tag2cbor(tagCborPrefix, out)
	n += arrayStart(out[n:])
	n += frame(stream, msg, out[n:])
	if err := t.tx(out[:n], false); err != nil {
		t.put(stream)
		return nil, err
	}
	return stream, nil
}

// | 0xd9f7 | packet2 |
func (t *Transport) stream(stream *Stream, msg Message, out []byte) error {
	n := tag2cbor(tagCborPrefix, out)
	n += frame(stream, msg, out[n:])
	err := t.tx(out[:n], false)
	if err != nil {
		t.put(stream)
		return err
	}
	return nil
}

// | 0xd9f7 | packetN | 0xff |
func (t *Transport) finish(stream *Stream, out []byte) error {
	n := tag2cbor(tagCborPrefix, out)
	n += breakStop(out[n:])
	err := t.tx(out[:n], false)
	t.put(stream)
	return err
}

func (t *Transport) frame(stream *Stream, msg Message, out []byte) (n int) {
	n += mapStart(out[n:])
	for key, value := range t.message(stream, msg) {
		n += value2cbor(key, out[n:])
		n += value2cbor(value, out[n:])
	}
	n += breakStop(out[n:])

	// create another buffer and rotate with `out` buffer and roll up the tags
	packeti = c.pktpool.Get()
	defer c.pktpool.Put(packeti)
	packet := packeti.([]byte)

	m := tag2cbor(tagMsg, packet) // roll up tagMsg
	m += copy(packet[m:], out[:n])

	for _, tag := range t.tags { // roll up tags
		if fn, ok := t.tagenc[tag]; ok { // only if requested by peer
			n = fn(packet[:m], out)
			m = tag2cbor(tag, packet)
			m += copy(packet[m:], out[:n])
		}
	}

	// finally roll up tagOpaqueStart-tagOpaqueEnd
	n = tag2cbor(stream.opaque, out)
	n += copy(out[:m], packet[:m])
	return n
}

func (t *Transport) message(stream *Stream, msg Message) map[string]interface{} {
	datai := t.pktpool.Get()
	defer t.pktpool.Put(datai)
	data := datai.([]byte)

	stream.blueprint[tagId] = msg.Id()
	p := msg.Encode(data)
	stream.blueprint[tagData] = data[:p]
	return stream.blueprint
}

var txprotopool = sync.Pool{New: func() interface{} { return &txreq{} }}

type txproto struct {
	packet []byte // request
	flush  bool
	count  int // response
	err    error
	respch chan *txproto
}

func (t *Transport) tx(packet []byte, flush bool) (n int, err error) {
	arg := txprotopool.Get().(*txproto)
	defer txprotopool.Put(arg)

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
