//  Copyright (c) 2014 Couchbase, Inc.

// Package gofast implements a high performance symmetric protocol
// for on the wire data transport.
//
// message-format:
//
//    A message is encoded as finite length CBOR map with predefined list
//    of keys, for example, "id", "data" etc... keys are typically encoded
//    as numbers so that they can be efficiently packed. This implies that
//    each of the predefined keys shall be assigned a unique number.
//
// an exchange shall be initiated either by client or server,
// exchange can be:
//
//     post-request, client post a packet and expects no response:
//
//          | 0xd9 0xd9f7 | packet |
//
//     request-response, client make a request and expects a
//     single response:
//
//          | 0xd9 0xd9f7 | 0x91 | packet |
//
//     bi-directional streaming, where client and server will have to close
//     the stream by sending a 0xff.
//
//          | 0xd9 0xd9f7  | 0x9f | packet1    |
//                 | 0xd9 0xd9f7  | packet2    |
//                 ...
//                 | 0xd9 0xd9f7  | end-packet |
//
//     * `packet` shall always be encoded as CBOR byte-array of info-type,
//       Info26 (4-byte) length.
//     * 0x91 denotes an array of single item, a special meaning for new
//       request that expects a single response from peer.
//     * 0x9f denotes an array of indefinite items, a special meaning
//       for a new stream that starts a bi-directional exchange.
//
// except of post-request, the exchange between client and server is always
// symmetrical.
//
// packet-format:
//
//    a single block of binary blob in CBOR format, transmitted
//    from client to server or server to client:
//
//       | tag1 |         payload1               |
//              | tag2 |      payload2           |
//                     | tag3 |   payload3       |
//                            | tag 4 | hdr-data |
//
//    * payload shall always be encoded as CBOR byte-array.
//    * hdr-data shall always be encoded as CBOR map.
//    * tags are uint64 numbers that will either be prefixed
//      to payload or msg.
//    * tag1, will always be a opaque number falling within a
//      reserved tag-space called opaque-space.
//    * tag2, tag3 can be one of the values predefined by gofast.
//    * the final embedded tag, in this case tag4, shall always
//      be tagMsg (value 37).
//
//      | tag1  | 0xff |
//
//    * if packet denotes a stream-end, payload will be
//      1-byte 0xff, and not encoded as byte-array.
//
//
// configurations:
//
//  "name"         - give a name for the transport.
//  "buffersize"   - maximum size that a packet can needs.
//  "chansize"     - channel size to use for internal go-routines.
//  "batchsize"    - number of packets to batch before writting to socket.
//  "tags"         - comma separated list of tags to apply, in specified order.
//  "opaque.start" - starting opaque range, inclusive.
//  "opaque.end"   - ending opaque range, inclusive.
//  "log.level"    - log level to use for DefaultLogger
//  "log.file"     - log file to use for DefaultLogger, if empty stdout is used.
//  "gzip.file"    - gzip compression level.
//
package gofast

import "sync"
import "sync/atomic"
import "fmt"
import "strings"
import "net"
import "time"

type tagfn func(in, out []byte) int

type RequestCallback func(*Stream, Message) chan Message

// Transporter interface to send and receive packets.
type Transporter interface { // facilitates unit testing
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close() error
}

// Transport is a peer-to-peer transport enabler.
type Transport struct {
	name     string
	version  Version
	peerver  Version
	tagenc   map[uint64]tagfn // tagid -> func
	tagdec   map[uint64]tagfn // tagid -> func
	strmpool chan *Stream
	messages map[uint64]Message // msgid -> message
	verfunc  func(interface{}) Version
	handlers map[uint64]RequestCallback

	conn    Transporter
	aliveat int64
	txch    chan *txproto
	rxch    chan interface{}
	killch  chan bool

	pktpool  *sync.Pool
	msgpools map[uint64]*sync.Pool

	config    map[string]interface{}
	logprefix string
}

//---- transport initialization APIs

// NewTransport creates a new transport over a connection,
// one connection one transport.
func NewTransport(
	conn Transporter,
	version Version,
	logg Logger,
	config map[string]interface{}) (*Transport, error) {

	name := config["name"].(string)
	buffersize := config["buffersize"].(int)
	opqstart := config["opaque.start"].(int)
	opqend := config["opaque.end"].(int)
	chansize := config["chansize"].(int)

	t := &Transport{
		name:     name,
		version:  version,
		tagenc:   make(map[uint64]tagfn),
		tagdec:   make(map[uint64]tagfn),
		strmpool: nil, // shall be initialized after setOpaqueRange() call
		messages: make(map[uint64]Message),
		verfunc:  nil,

		conn:   conn,
		txch:   make(chan *txproto, chansize),
		rxch:   make(chan interface{}, chansize),
		killch: make(chan bool),

		msgpools: make(map[uint64]*sync.Pool),

		config: config,
	}

	setLogger(logg, t.config)

	laddr, raddr := conn.LocalAddr(), conn.RemoteAddr()
	t.logprefix = fmt.Sprintf("GFST[%v<->%v]", laddr, raddr)
	t.pktpool = &sync.Pool{
		New: func() interface{} { return make([]byte, buffersize) },
	}
	t.setOpaqueRange(uint64(opqstart), uint64(opqend))
	t.SubscribeMessages(t.msghandler, &Whoami{}, &Ping{}, &Heartbeat{})

	// educate tranport with configured tag decoders.
	tagcsv, _ := config["tags"]
	for _, tag := range t.getTags(tagcsv.(string), []string{}) {
		if factory, ok := tag_factory[tag]; ok {
			tagid, _, dec := factory(t, config)
			t.tagdec[tagid] = dec
		}
		panic(fmt.Errorf("unknown tag %v", tag))
	}
	log.Verbosef("%v pre-initialized ...\n", t.logprefix)

	go t.doTx()
	go t.syncRx() // will spawn another go-routine doRx().

	msg, err := t.Whoami()
	if err != nil {
		return nil, err
	}
	// parse tag list, tags will be applied in the specified order.
	for _, tag := range t.getTags(msg.tags, []string{}) {
		if factory, ok := tag_factory[tag]; ok {
			tagid, enc, _ := factory(t, config)
			t.tagenc[tagid] = enc
		}
		log.Warnf("%v remote ask for unknown tag: %v", t.logprefix, tag)
	}

	fmsg := "%v handshake completed with peer: %v ...\n"
	log.Verbosef(fmsg, t.logprefix, msg.Repr())

	log.Infof("%v started ...\n", t.logprefix)
	return t, nil
}

// VersionHandler callback to convert peer version to Version interface.
func (t *Transport) VersionHandler(fn func(value interface{}) Version) *Transport {
	t.verfunc = fn
	return t
}

// SubscribeMessages that shall be exchanged via this transport. Only
// subscribed messages can be exchanged.
func (t *Transport) SubscribeMessages(handler RequestCallback, msgs ...Message) *Transport {
	msgstr := []string{}
	for _, msg := range msgs {
		id := msg.Id()
		if isReservedMsg(id) {
			panic(fmt.Errorf("message id %v reserved", id))
		}
		t.messages[id] = msg
		t.msgpools[id] = &sync.Pool{New: objfactory(msg)}
		t.handlers[id] = handler
		msgstr = append(msgstr, msg.String())
	}
	s := strings.Join(msgstr, ",")
	log.Verbosef("%v subscribed %v\n", t.logprefix, s)
	return t
}

// Close this tranport, connection will be closed as well.
func (t *Transport) Close() error {
	defer func() { recover() }()
	// closing kill-channel should accomplish the following,
	// a. prevent any more transmission on the connection.
	// b. close all active streams.
	close(t.killch)
	// drain out the stream-pool.
	for range t.strmpool {
	}
	t.pktpool = nil
	log.Infof("%v ... closed\n", t.logprefix)
	// finally close the connection itself.
	return t.conn.Close()
}

//---- maintenance APIs

// FlushPeriod to periodically flush batched packets.
func (t *Transport) FlushPeriod(ms time.Duration) {
	now, tick := time.Now(), time.Tick(ms)
	go func() {
		for {
			<-tick
			t.tx([]byte{} /*empty*/, true /*flush*/)
			log.Debugf("%v flushed after %v\n", t.logprefix, time.Since(now))
		}
	}()
}

// SendHeartbeat to periodically send keep-alive message.
func (t *Transport) SendHeartbeat(ms time.Duration) {
	count, tick := uint64(0), time.Tick(ms)
	go func() {
		for {
			<-tick
			msg := NewHeartbeat(t, count)
			t.Post(msg)
			t.Free(msg)
			count++
			log.Debugf("%v posted heartbeat %v\n", t.logprefix, count)
		}
	}()
}

// Silentsince returns the timestamp of last heartbeat message received
// from peer.
func (t *Transport) Silentsince() time.Duration {
	then := time.Unix(0, atomic.LoadInt64(&t.aliveat))
	return time.Since(then)
}

// LocalAddr of this connection.
func (t *Transport) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

// RemoteAddr of this connection.
func (t *Transport) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

// PeerVersion from peer node.
func (t *Transport) PeerVersion() Version {
	return t.peerver
}

// Free will return the message back to the pool, should be
// called on messages received via RequestCallback callback and
// via Stream.Rxch.
func (t *Transport) Free(msg Message) {
	t.msgpools[msg.Id()].Put(msg)
}

//---- transport APIs

// Whoami will return remote information.
func (t *Transport) Whoami() (*Whoami, error) {
	msg := NewWhoami(t)
	defer t.Free(msg)
	resp, err := t.Request(msg)
	if err != nil {
		return nil, err
	}
	return resp.(*Whoami), nil
}

// Ping pong with peer.
func (t *Transport) Ping(echo string) (*Ping, error) {
	msg := NewPing(echo)
	defer t.Free(msg)
	resp, err := t.Request(msg)
	if err != nil {
		return nil, err
	}
	return resp.(*Ping), nil
}

// Post request to peer.
func (t *Transport) Post(msg Message) error {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)
	stream := t.getstream(nil)
	defer t.putstream(stream.opaque, stream, true /*tellrx*/)

	n := t.post(msg, stream, out)
	return t.tx(out[:n], false)
}

// Request a response from peer.
func (t *Transport) Request(msg Message) (response Message, err error) {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)
	stream := t.getstream(make(chan Message, 1))
	defer t.putstream(stream.opaque, stream, true /*tellrx*/)

	var ok bool
	n := t.request(msg, stream, out)
	if err = t.tx(out[:n], false); err == nil {
		if response, ok = <-stream.Rxch; ok {
			return
		}
		err = fmt.Errorf("stream closed")
	}
	return
}

// Request a bi-directional with peer.
func (t *Transport) Stream(msg Message, ch chan Message) (stream *Stream, err error) {
	out := t.pktpool.Get().([]byte)
	defer t.pktpool.Put(out)

	stream = t.getstream(ch)
	defer func() {
		if err != nil {
			t.putstream(stream.opaque, stream, true /*tellrx*/)
		}
	}()
	n := t.start(msg, stream, out)
	err = t.tx(out[:n], false)
	return
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
	log.Debugf("%v local streams start %v end %v\n", t.logprefix, start, end)
	t.strmpool = make(chan *Stream, end-start+1) // inclusive
	for opaque := start; opaque <= end; opaque++ {
		t.strmpool <- t.newstream(uint64(opaque), false)
	}
}

func (t *Transport) getTags(line string, tags []string) []string {
	for _, tag := range strings.Split(line, ",") {
		if strings.Trim(tag, " \n\t\r") != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

type tagFactory func(*Transport, map[string]interface{}) (uint64, tagfn, tagfn)

var tag_factory = make(map[string]tagFactory)
