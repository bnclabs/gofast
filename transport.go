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

type tagfn func(in, out []byte) int

// Transporter interface to send and receive packets.
type Transporter interface { // facilitates unit testing
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type Transport struct {
	cbor      *Config
	name      string
	version   Version
	peerver   Version
	tagenc    map[uint64]tagfn // tagid -> func
	tagdec    map[uint64]tagfn // tagid -> func
	tagok     map[uint64]bool  // tagid -> bool, requested while handshake
	streams   chan *stream
	messages  map[uint64]Message // msgid -> message
	verfunc   func(value, interface{}) Version
	blueprint map[uint64]interface{} // message-tags -> value

	conn Transporter
	txch chan *txproto
	rxch chan interface{}

	pktpool *sync.Pool

	config map[string]interface{}
	tags   []string
}

//---- transport initialization APIs

func NewTransport(conn Transporter, version Version, config map[string]interface{}) *Transport {
	name := config["name"].(string)
	buflen := config["buflen"].(int)
	opqstart := config["opaque.start"].(int)
	opqend := config["opaque.end"].(int)
	_ := config["batchlen"].(int)

	t := Transport{
		cbor:      NewDefaultConfig(),
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
		txch: make(chan *txproto, 1024),

		config: config,
		tags:   make([]string, 0, 8),
	}
	t.pktpool = sync.Pool{
		New: func() interface{} { return make([]byte, buflen) },
	}
	// parse tag list, tags will be applied in the specified order
	// do this before invoking tag-factory
	if tagcsv, ok := config["tags"]; ok {
		for _, tag := range strings.Split(tagcsv, ",") {
			if strings.Trim(tag) != "" {
				t.tags = append(t.tags, tag)
			}
		}
	}
	// pre-initialize blueprint
	t.blueprint[tagId] = nil
	t.blueprint[tagData] = nil

	t.setOpaqueRange(opqstart, opqend) // precise sequence

	t.SubscribeMessages([]Message{&Whoami{}, &Ping{}, &Heartbeat{}})
	go doTx()

	t.Ping("handshake")
	t.Whoami()

	// initialize blueprint and tag-wrapping
	t.blueprint[tagVersion] = t.peerver.Value()
	for tag, factory := range tag_factory {
		enc, dec := factory(t, config)
		if enc == nil || dec == nil {
			continue
		}
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
	count, tick := 0, time.Tick(ms)
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

func (t *Transport) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *Transport) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

//---- transport APIs

func (t *Transport) Whoami() *Whoami {
	msg := NewWhoami()
	resp := t.Request(msg).(*Whoami)
	msg.Free()
	return resp.(*Whoami)
}

func (t *Transport) Ping(echo string) *Ping {
	msg := NewPing()
	resp := t.Request(msg)
	msg.Free()
	return resp.(*Ping)
}

func (t *Transport) Post(msg Message) {
	t.post(msg, out)
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
	t.streams = make(chan *streams, end-start+1) // inclusive
	for opaque := start; opaque <= end; opaque++ {
		stream := t.newstream(uint64(opaque))
		stream = t.cloneblueprint()
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

var tag_factory = make(map[uint64]func(*Transport, map[string]interface{}) (tagfn, tagfn))
