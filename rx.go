package gofast

var rxpool = sync.Pool{New: func() interface{} { return &rxpacket{} }}

type rxpacket struct {
	packet []byte
	opaque uint64
}

func (t *Transport) syncrx() {
	tagrunners := make(map[uint64]chan []byte)
	tagrunners[typMsg] = rxch
	for _, tag := range t.tags {
		inch := make(chan *rxpacket, 0)
		tagrunners[tag] = inch
		go func(tag uint64, inch chan []byte) { // this is the tagrunner
			for {
				func() {
					rxpkt := <-inch
					defer t.pktpool.Put(rxpkt.packet)

					packet = t.pktpool.Get().([]byte)
					ln = t.tagdec[tag](rxpkt.packet, packet)
					packet = packet[:ln]

					ntag, n := cbor2tag(packet)
					_, m := cborItemLength(packet[n:])
					n += m
					n = copy(packet, packet[n:])
					rxpkt.packet = packet[:n]
					tagrunners[ntag.(uint64)] <- rxpkt
				}()
			}
		}()
	}

	streams := make(map[uint64]*Stream)
	for {
		arg := <-rxch
		switch val := arg.(type) {
		case *Stream:
			_, ok := streams[val.opaque]
			if val.Rxch == nil && ok {
				delete(streams, val.opaque)
			} else {
				streams[val.opaque] = val
			}

		case *rxpkt:
			msg := t.unmessage(rxpkt.packet)
			streams[opaque].Rxch <- msg
			t.pktpool.Put(rxpkt.packet)
			rxpool.Put(rxpkt)
		}
	}
}

func (t *Transport) doRx(tagrunners) {
	var scratch [9]byte

	downstream := func(packet []byte) error {
		if n, err := io.ReadFull(t.conn, packet); err != nil {
			return err
		} else if n < size {
			return err // TODO: create some valid error
		}
		tag, n := cbor2tag(packet)
		opaque := tag.(uint64)
		_, m := cborItemLength(packet[n:])
		n += m
		tag, m = cbor2tag(packet[n:])
		n += m
		_, m := cborItemLength(packet[n:])
		n += m
		n = copy(packet, packet[n:])
		tagrunners[tag.(uint64)] <- &rxpacket{
			opaque: opaque,
			packet: packet[:n],
		}
		return nil
	}

	for {
		n, err := io.ReadFull(t.conn, scratch[:3]) // TODO: handle err
		packet := t.pktpool.Get().([]byte)
		if scratch[0] != 0xf7 || scratch[1] != 0xd9 { // check cbor-prefix
			// TODO: handle err
		} else if scratch[2] == 0xff {
			// TODO: handle end of stream
		} else if info := cborInfo(scratch[2]); info < cborInfo24 {
			err := downstream(packet[:info]) // TODO: handle err
		} else if info == cborInfo24 {
			n, err := io.ReadFull(scratch[:1])
			size, _ := cbor2valt0info24(scratch[:1])
			err := downstream(packet[:size]) // TODO: handle err
		} else if info == cborInfo25 {
			n, err := io.ReadFull(scratch[:2])
			size, _ := cbor2valt0info25(scratch[:2])
			err := downstream(packetx[:size]) // TODO: handle err
		} else if info == cborInfo26 {
			n, err := io.ReadFull(scratch[:4])
			size, _ := cbor2valt0info26(scratch[:4])
			err := downstream(packet[:size]) // TODO: handle err
		} else if info == cborInfo27 {
			n, err := io.ReadFull(scratch[:8])
			size, _ := cbor2valt0info27(scratch[:8])
			err := downstream(packet[:size]) // TODO: handle err
		} else {
			// TODO: handle err
		}
	}
}

func (t *Transport) unmessage(msg []byte) Message {
	if msg[n] != 0xbf {
		panic("expected a cbor map for message")
	}
	n++

	typ, data := uint64(0), nil
	for msg[n] != 0xff {
		key, n1 := cbor2value(msg[n:], c)
		value, n2 := cbor2value(msg[n+n1:], c)
		tag := key.(uint64)
		switch tag {
		case tagId:
			typ = tag
		case tagVersion:
			t.peerver = t.verfunc(value)
		case tagData:
			data = value.([]byte)
		default:
			if hasString(tag, t.tags) == false {
				log.Warnf("unknown tag requested %v\n", tag)
			} else {
				t.tagok[tag] = true
			}
		}
		n += n1 + n2
	}
	// check whether type, data is present
	if typ == 0 || data == nil {
		log.Errorf("rx invalid message packet")
		return nil
	}
	typeOfMsg := reflect.ValueOf(t.messages[typ]).Elem().Type()
	msg = reflect.New(typeOfMsg).Interface().(Message)
	msg.Decode(data)
	return msg
}
