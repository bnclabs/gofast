package gofast

import "sync"
import "io"
import "fmt"
import "sync/atomic"
import "time"

var rxpool = sync.Pool{New: func() interface{} { return &rxpacket{} }}

type rxpacket struct {
	packet []byte
	opaque uint64
	finish bool
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

	fmsg := "%v syncRx(%v, %v) started ...\n"
	log.Infof(fmsg, t.logprefix, chansize, t.tags)
	livestreams := make(map[uint64]*Stream)
loop:
	for {
		select {
		case arg := <-t.rxch:
			switch val := arg.(type) {
			case *Stream:
				_, ok := livestreams[val.opaque]
				if val.Rxch == nil && ok {
					fmsg := "%v stream ##%x closed ...\n"
					log.Debugf(fmsg, t.logprefix, val.opaque)
					delete(livestreams, val.opaque)
				} else {
					fmsg := "%v stream ##%x started ...\n"
					log.Debugf(fmsg, t.logprefix, val.opaque)
					livestreams[val.opaque] = val
				}

			case *rxpacket:
				stream, streamok := livestreams[val.opaque]
				msg := t.unmessage(val.packet)
				switch knownmsg := msg.(type) {
				case *Heartbeat:
					atomic.StoreInt64(&t.liveat, time.Now().UnixNano())
				case *Ping:
					if streamok {
						stream.Rxch <- knownmsg
					} else if _, err := t.Ping(knownmsg.echo); err != nil {
						log.Errorf("%v ping: %v\n", t.logprefix, err)
					}
				case *Whoami:
					if streamok {
						stream.Rxch <- knownmsg
					} else if _, err := t.Whoami(); err != nil {
						log.Errorf("%v whoami: %v\n", t.logprefix, err)
					}
				case Message:
					if streamok {
						stream.Rxch <- knownmsg
					} else { // new request from peer
						stream = t.newstream(val.opaque)
						stream.remote, stream.Rxch = true, nil
						fmsg := "%v remote stream ##%x started ...\n"
						log.Debugf(fmsg, t.logprefix, stream.opaque)
						livestreams[stream.opaque] = stream
						stream.Rxch = t.handler(stream, knownmsg)
					}
				default:
					fmsg := "%v stream %x not expected\n"
					log.Errorf(fmsg, t.logprefix, val.opaque)
				}
				if streamok && val.finish {
					fmsg := "%v stream ##%x closed by remote ...\n"
					log.Debugf(fmsg, t.logprefix, stream.opaque)
					t.putstream(stream)
					close(stream.Rxch)
					stream.Rxch = nil
					delete(livestreams, val.opaque)
					if stream.remote == false { // reclaim if local stream
						t.streams <- stream
					}
				}
				t.pktpool.Put(val.packet)
				rxpool.Put(val)

			case string:
				switch val {
				case "close":
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
	log.Infof("%v syncRx() ... stopped\n")
}

func (t *Transport) doRx(tagrunners map[uint64]chan interface{}) {
	defer func() { go t.Close() }()

	downstream := func(packet []byte) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("%v malformed packet: %v\n", t.logprefix, r)
				err = fmt.Errorf("%v", r)
			}
		}()

		if n, err := io.ReadFull(t.conn, packet); err != nil {
			return err
		} else if n < len(packet) {
			return fmt.Errorf("partial write (%v)", n)
		}

		rxpkt := rxpool.Get().(*rxpacket)

		tag, n := cbor2tag(packet)
		rxpkt.opaque = tag.(uint64)
		_, m := cborItemLength(packet[n:])
		n += m
		tag, m = cbor2tag(packet[n:])
		n += m
		_, m = cborItemLength(packet[n:])
		n += m
		n = copy(packet, packet[n:]) // overwrite opaque,len,tag,len
		rxpkt.packet = packet[:n]
		if packet[n] == 0xff {
			rxpkt.finish = true
		}
		tagrunners[tag.(uint64)] <- rxpkt
		return nil
	}
	log.Infof("%v doRx() started ...\n", t.logprefix)
	var pad [9]byte
	for {
		if n, err := io.ReadFull(t.conn, pad[:3]); err != nil || n != 3 {
			log.Errorf("%v reading prefix: %v,%v\n", t.logprefix, n, err)
			break
		}
		packet := t.pktpool.Get().([]byte)
		if pad[0] != 0xf7 || pad[1] != 0xd9 { // check cbor-prefix
			log.Errorf("%v wrong prefix\n", t.logprefix)
			break
		} else if info := cborInfo(pad[2]); info < cborInfo24 {
			if err := downstream(packet[:info]); err != nil {
				log.Errorf("%v reading small packet: %v\n", t.logprefix, err)
				break
			}
		} else if info == cborInfo24 {
			if n, err := io.ReadFull(t.conn, pad[:1]); n != 1 || err != nil {
				fmsg := "%v reading cborInfo24: %v,%v\n"
				log.Errorf(fmsg, t.logprefix, n, err)
				break
			}
			size, _ := cbor2valt0info24(pad[:1])
			if err := downstream(packet[:size.(uint64)]); err != nil {
				log.Errorf("%v reading typical packet: %v\n", t.logprefix, err)
				break
			}
		} else if info == cborInfo25 {
			if n, err := io.ReadFull(t.conn, pad[:2]); n != 2 || err != nil {
				fmsg := "%v reading cborInfo25: %v,%v\n"
				log.Errorf(fmsg, t.logprefix, n, err)
				break
			}
			size, _ := cbor2valt0info25(pad[:2])
			if err := downstream(packet[:size.(uint64)]); err != nil {
				log.Errorf("%v reading medium packet: %v\n", t.logprefix, err)
				break
			}
		} else if info == cborInfo26 {
			if n, err := io.ReadFull(t.conn, pad[:4]); n != 4 || err != nil {
				log.Errorf("%v reading cborInfo26: %v,%v\n", t.logprefix, n, err)
				break
			}
			size, _ := cbor2valt0info26(pad[:4])
			if err := downstream(packet[:size.(uint64)]); err != nil {
				log.Errorf("%v reading large packet: %v\n", t.logprefix, err)
				break
			}
		} else if info == cborInfo27 {
			if n, err := io.ReadFull(t.conn, pad[:8]); n != 8 || err != nil {
				fmsg := "%v reading cborInfo27: %v,%v\n"
				log.Errorf(fmsg, t.logprefix, n, err)
				break
			}
			size, _ := cbor2valt0info27(pad[:8])
			if err := downstream(packet[:size.(uint64)]); err != nil {
				fmsg := "%v reading very-large packet: %v\n"
				log.Errorf(fmsg, t.logprefix, err)
				break
			}
		} else {
			log.Errorf("%v unknown packet len\n", t.logprefix)
			break
		}
	}
	log.Infof("%v doRx() ... stopped\n", t.logprefix)
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
				log.Warnf("%v unknown tag requested %v\n", t.logprefix, tag)
			}
		}
		n += n1 + n2
	}
	// check whether id, data is present
	if id == 0 || data == nil {
		log.Errorf("%v rx invalid message packet\n", t.logprefix)
		return nil
	}
	msg := t.msgpools[id].Get().(Message)
	msg.Decode(data)
	return msg
}
