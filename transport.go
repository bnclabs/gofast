//  Copyright (c) 2014 Couchbase, Inc.

// Package gofast implements a high performance symmetric protocol
// for on the wire data transport.
//
// message-format:
//
//	  A message is encoded as finite length CBOR map with predefined list
//    of keys, for example, "version", "compression", "id", "data" etc...
//    keys are typically encoded as uint16 numbers so that
//    they can be efficiently packed. This implies that each of the
//    predefined keys shall be assigned a unique number.
//
// packet-format:
//
//	  a single block of binary blob in CBOR format, transmitted
//    from client to server or server to client:
//
//		tag1 |         payload1				  |
//			 | tag2 |      payload2			  |
//					| tag3 |   payload3       |
//						   | tag 4 | hdr-data |
//
//
//    * payload shall always be encoded as CBOR byte array.
//	  * tags are uint64 numbers that will either be prefixed
//      to payload or msg.
//    * packets are encoded as cbor-bytes.
//
//    In the above example:
//
//	  * tag1, will always be a opaque number falling within a
//      reserved tag-space called opaque-space.
//    * tag2, tag3 can be one of the values predefined by gofast.
//    * the final embedded tag, in this case tag4, shall always
//      be tagMsg (value 37).
//
// an exchange shall be initiated either by client or server,
// exchange can be:
//
//	   post-request, client post a packet and expects no response:
//
//			| 0xd9f7 | packet |
//
//	   request-response, client make a request and expects a
//     single response:
//
//			| 0xd9f7 | 0x9f | packet | 0xff |
//
//	   bi-directional streaming, where client and server will have to close
//     the stream by sending a 0xff.
//
//			| 0xd9f7 | 0x9f | packet1 |
//							| 0xd9f7 | packet2 |
//							...
//							| 0xd9f7 | packetN | 0xff |
//
// except of post-request, the exchange between client and server is always
// symmetrical.
package gofast

import "sync"
import "sync/atomic"
import "fmt"
import "strings"
import "net"
import "time"

type tagfn func(in, out []byte) int

// Transporter interface to send and receive packets.
type Transporter interface { // facilitates unit testing
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close() error
}

type Transport struct {
	name      string
	version   Version
	peerver   Version
	tagenc    map[uint64]tagfn // tagid -> func
	tagdec    map[uint64]tagfn // tagid -> func
	tagok     map[uint64]bool  // tagid -> bool, requested while handshake
	streams   chan *Stream
	messages  map[uint64]Message // msgid -> message
	verfunc   func(interface{}) Version
	blueprint map[uint64]interface{} // message-tags -> value

	conn   Transporter
	liveat int64
	txch   chan *txproto
	rxch   chan interface{}
	killch chan bool

	pktpool *sync.Pool

	config map[string]interface{}
	tags   map[uint64]bool
}

//---- transport initialization APIs

func NewTransport(conn Transporter, version Version, config map[string]interface{}) *Transport {
	name := config["name"].(string)
	buflen := config["buflen"].(int)
	opqstart := config["opaque.start"].(int)
	opqend := config["opaque.end"].(int)
	chansize := config["chansize"].(int)
	setLogger(nil, config)

	t := &Transport{
		name:      name,
		version:   version,
		tagenc:    make(map[uint64]tagfn),
		tagok:     make(map[uint64]bool),
		tagdec:    make(map[uint64]tagfn),
		streams:   nil, // shall be initialized after SetOpaqueRange() call
		messages:  make(map[uint64]Message),
		verfunc:   nil,
		blueprint: make(map[uint64]interface{}),

		conn: conn,
		txch: make(chan *txproto, chansize),
		rxch: make(chan interface{}, chansize),

		config: config,
		tags:   make(map[uint64]bool),
	}
	t.pktpool = &sync.Pool{
		New: func() interface{} { return make([]byte, buflen) },
	}
	// pre-initialize blueprint
	t.blueprint[tagId] = nil
	t.blueprint[tagData] = nil

	t.setOpaqueRange(uint64(opqstart), uint64(opqend)) // precise sequence

	t.SubscribeMessages([]Message{&Whoami{}, &Ping{}, &Heartbeat{}})
	go t.doTx()
	go t.syncRx()

	t.Ping("handshake")
	t.Whoami()

	// parse tag list, tags will be applied in the specified order
	// do this before invoking tag-factory
	tagmap := map[string]bool{}
	if tagcsv, ok := config["tags"].(string); ok {
		for _, tagname := range strings.Split(tagcsv, ",") {
			if strings.Trim(tagname, " \n\t\r") != "" {
				tagmap[tagname] = true
			}
		}
	}
	// initialize blueprint and tag-wrapping
	t.blueprint[tagVersion] = t.peerver.Value()
	for tagname, factory := range tag_factory {
		if _, ok := tagmap[tagname]; !ok {
			continue
		}
		tag, enc, dec := factory(t, config)
		t.tags[tag] = true
		t.tagenc[tag] = enc
		t.tagdec[tag] = dec
		t.blueprint[tag] = nil
	}
	// since blue-print is modified, reinitialize the blue-prints.
	streams := make([]*Stream, 0, len(t.streams))
	for stream := range t.streams {
		stream.blueprint = t.cloneblueprint()
		streams = append(streams, stream)
	}
	for _, stream := range streams {
		t.streams <- stream
	}
	return t
}

func (t *Transport) SetLogger(logger Logger) *Transport {
	setLogger(logger, t.config)
	return t
}

func (t *Transport) VersionHandler(fn func(value interface{}) Version) *Transport {
	t.verfunc = fn
	return t
}

func (t *Transport) SubscribeMessages(messages []Message) *Transport {
	for _, msg := range messages {
		t.messages[msg.Id()] = msg
	}
	return t
}

func (t *Transport) Close() error {
	defer func() { recover() }()
	t.pktpool = nil
	t.rxch <- "close"
	close(t.killch)
	return t.conn.Close()
}

//---- maintenance APIs

func (t *Transport) FlushPeriod(ms time.Duration) {
	tick := time.Tick(ms)
	go func() {
		for {
			<-tick
			t.tx([]byte{} /*empty*/, true /*flush*/)
		}
	}()
}

func (t *Transport) SendHeartbeat(ms time.Duration) {
	count, tick := uint64(0), time.Tick(ms)
	go func() {
		for {
			<-tick
			msg := NewHeartbeat(t, count)
			t.Post(msg)
			msg.Free()
			count++
		}
	}()
}

func (t *Transport) Liveat() int64 {
	return atomic.LoadInt64(&t.liveat)
}

func (t *Transport) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *Transport) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

//---- transport APIs

func (t *Transport) Whoami() (*Whoami, error) {
	msg := NewWhoami(t)
	defer msg.Free()
	resp, err := t.Request(msg)
	if err != nil {
		return nil, err
	}
	return resp.(*Whoami), nil
}

func (t *Transport) Ping(echo string) (*Ping, error) {
	msg := NewPing(echo)
	defer msg.Free()
	resp, err := t.Request(msg)
	if err != nil {
		return nil, err
	}
	return resp.(*Ping), nil
}

func (t *Transport) Post(msg Message) error {
	return t.post(msg)
}

func (t *Transport) Request(msg Message) (Message, error) {
	return t.request(msg)
}

func (t *Transport) Stream(msg Message, ch chan Message) (*Stream, error) {
	return t.start(msg, ch)
}

//---- local APIs

func (t *Transport) setOpaqueRange(start, end uint64) {
	fmsg := "opaques must start `%v` and shall not exceed `%v`"
	err := fmt.Errorf(fmsg, tagOpaqueStart, tagOpaqueEnd)
	if start != tagOpaqueStart {
		panic(err)
	} else if end > tagOpaqueEnd {
		panic(err)
	}
	t.streams = make(chan *Stream, end-start+1) // inclusive
	for opaque := start; opaque <= end; opaque++ {
		stream := t.newstream(uint64(opaque))
		stream.blueprint = t.cloneblueprint()
		t.streams <- stream
	}
}

func (t *Transport) cloneblueprint() map[uint64]interface{} {
	// once transport is initialized, blueprint is immutable.
	blueprint := make(map[uint64]interface{})
	blueprint[tagId] = nil
	// during pre-initialization version won't be present.
	if version, ok := t.blueprint[tagVersion]; ok {
		blueprint[tagVersion] = version
	}
	blueprint[tagData] = nil
	for tag, _ := range t.blueprint {
		blueprint[tag] = nil
	}
	return blueprint
}

var tag_factory = make(map[string]func(*Transport, map[string]interface{}) (uint64, tagfn, tagfn))
