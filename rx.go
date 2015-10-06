package gofast

import "sync"
import "io"
import "fmt"
import "sync/atomic"
import "time"
import "reflect"

var rxpool = sync.Pool{New: func() interface{} { return &rxpacket{} }}

type rxpacket struct {
	packet []byte
	opaque uint64
}

func (t *Transport) syncRx() {
	chansize := t.config["chansize"].(int)
	tagrunners := make(map[uint64]chan interface{})
	tagrunners[tagMsg] = t.rxch
	for tag := range t.tags {
		inch := make(chan interface{}, chansize)
		tagrunners[tag] = inch
		go func(tag uint64, inch chan interface{}) { // this is the tagrunner
			for {
				func() {
					arg := <-inch
					rxpkt := arg.(*rxpacket)
					defer t.pktpool.Put(rxpkt.packet)

					packet := t.pktpool.Get().([]byte)
					n := t.tagdec[tag](rxpkt.packet, packet)
					packet = packet[:n]

					ntag, n := cbor2tag(packet)
					_, m := cborItemLength(packet[n:])
					n += m
					n = copy(packet, packet[n:]) // overwrite tag,len
					rxpkt.packet = packet[:n]
					tagrunners[ntag.(uint64)] <- rxpkt
				}()
			}
		}(tag, inch)
	}

	go t.doRx(tagrunners)

	livestreams := make(map[uint64]*Stream)
loop:
	for {
		select {
		case arg := <-t.rxch:
			switch val := arg.(type) {
			case *Stream:
				_, ok := livestreams[val.opaque]
				if val.Rxch == nil && ok {
					delete(livestreams, val.opaque)
				} else {
					livestreams[val.opaque] = val
				}

			case *rxpacket:
				msg := t.unmessage(val.packet)
				switch knownmsg := msg.(type) {
				case *Heartbeat:
					atomic.StoreInt64(&t.liveat, time.Now().UnixNano())
				case *Ping:
					if _, err := t.Ping(knownmsg.echo); err != nil {
						log.Errorf("ping error: %v", err)
					}
				case *Whoami:
					if _, err := t.Whoami(); err != nil {
						log.Errorf("whoami error: %v", err)
					}
				default:
					livestreams[val.opaque].Rxch <- msg
				}
				t.pktpool.Put(val.packet)
				rxpool.Put(val)

			case string:
				if val == "close" {
					for _, stream := range livestreams {
						stream.Close()
					}
					close(t.txch)
				}
			}

		case <-t.killch:
			break loop
		}
	}
}

func (t *Transport) doRx(tagrunners map[uint64]chan interface{}) {
	defer func() { go t.Close() }()

	downstream := func(packet []byte) error {
		if n, err := io.ReadFull(t.conn, packet); err != nil {
			return err
		} else if n < len(packet) {
			return fmt.Errorf("partial write (%v)", n)
		}
		tag, n := cbor2tag(packet)
		opaque := tag.(uint64)
		_, m := cborItemLength(packet[n:])
		n += m
		tag, m = cbor2tag(packet[n:])
		n += m
		_, m = cborItemLength(packet[n:])
		n += m
		n = copy(packet, packet[n:]) // overwrite opaque,len,tag,len
		rxpkt := rxpool.Get().(*rxpacket)
		rxpkt.opaque = opaque
		rxpkt.packet = packet[:n]
		tagrunners[tag.(uint64)] <- rxpkt
		return nil
	}

	var pad [9]byte
	for {
		if n, err := io.ReadFull(t.conn, pad[:3]); err != nil || n != 3 {
			log.Errorf("reading prefix: %v,%v", n, err)
			break
		}
		packet := t.pktpool.Get().([]byte)
		if pad[0] != 0xf7 || pad[1] != 0xd9 { // check cbor-prefix
			log.Errorf("wrong prefix")
			break
		} else if pad[2] == 0xff {
			// TODO: handle end of stream
		} else if info := cborInfo(pad[2]); info < cborInfo24 {
			if err := downstream(packet[:info]); err != nil {
				log.Errorf("reading small packet: %v", err)
				break
			}
		} else if info == cborInfo24 {
			if n, err := io.ReadFull(t.conn, pad[:1]); n != 1 || err != nil {
				log.Errorf("reading cborInfo24: %v,%v", n, err)
				break
			}
			size, _ := cbor2valt0info24(pad[:1])
			if err := downstream(packet[:size.(uint64)]); err != nil {
				log.Errorf("reading typical packet: %v", err)
				break
			}
		} else if info == cborInfo25 {
			if n, err := io.ReadFull(t.conn, pad[:2]); n != 2 || err != nil {
				log.Errorf("reading cborInfo25: %v,%v", n, err)
				break
			}
			size, _ := cbor2valt0info25(pad[:2])
			if err := downstream(packet[:size.(uint64)]); err != nil {
				log.Errorf("reading medium packet: %v", err)
				break
			}
		} else if info == cborInfo26 {
			if n, err := io.ReadFull(t.conn, pad[:4]); n != 4 || err != nil {
				log.Errorf("reading cborInfo26: %v,%v", n, err)
				break
			}
			size, _ := cbor2valt0info26(pad[:4])
			if err := downstream(packet[:size.(uint64)]); err != nil {
				log.Errorf("reading large packet: %v", err)
				break
			}
		} else if info == cborInfo27 {
			if n, err := io.ReadFull(t.conn, pad[:8]); n != 8 || err != nil {
				log.Errorf("reading cborInfo27: %v,%v", n, err)
				break
			}
			size, _ := cbor2valt0info27(pad[:8])
			if err := downstream(packet[:size.(uint64)]); err != nil {
				log.Errorf("reading very-large packet: %v", err)
				break
			}
		} else {
			log.Errorf("unknown packet len")
			break
		}
	}
}

func (t *Transport) unmessage(msgdata []byte) Message {
	n := 0
	if msgdata[n] != 0xbf {
		panic("expected a cbor map for message")
	}
	n++

	var id uint64
	var data []byte
	for msgdata[n] != 0xff {
		key, n1 := cbor2value(msgdata[n:])
		value, n2 := cbor2value(msgdata[n+n1:])
		tag := key.(uint64)
		switch tag {
		case tagId:
			id = tag
		case tagVersion:
			t.peerver = t.verfunc(value)
		case tagData:
			data = value.([]byte)
		default:
			if b, ok := t.tags[tag]; ok {
				t.tagok[tag] = b
			} else {
				log.Warnf("unknown tag requested %v\n", tag)
			}
		}
		n += n1 + n2
	}
	// check whether id, data is present
	if id == 0 || data == nil {
		log.Errorf("rx invalid message packet")
		return nil
	}
	typeOfMsg := reflect.ValueOf(t.messages[id]).Elem().Type()
	msg := reflect.New(typeOfMsg).Interface().(Message)
	msg.Decode(data)
	return msg
}
