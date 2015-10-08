//  Copyright (c) 2014 Couchbase, Inc.

// Package gofast implements a high performance symmetric protocol
// for on the wire data transport.
//
// message-format:
//
//	  A message is encoded as finite length CBOR map with predefined list
//    of keys, for example, "id", "data" etc... keys are typically encoded
//    as numbers so that they can be efficiently packed. This implies that
//    each of the predefined keys shall be assigned a unique number.
//
// an exchange shall be initiated either by client or server,
// exchange can be:
//
//	   post-request, client post a packet and expects no response:
//
//			| 0xd9 0xd9f7 | packet |
//
//	   request-response, client make a request and expects a
//     single response:
//
//			| 0xd9 0xd9f7 | 0x91 | packet |
//
//	   bi-directional streaming, where client and server will have to close
//     the stream by sending a 0xff.
//
//			| 0xd9 0xd9f7 | 0x9f | packet1 |
//							| 0xd9 0xd9f7  | packet2     |
//							...
//							| 0xd9 0xd9f7  | end-packet  |
//
//	   * `packet` shall always be encoded as CBOR byte-array of info-type,
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
//	  a single block of binary blob in CBOR format, transmitted
//    from client to server or server to client:
//
//       | tag1 |         payload1               |
//              | tag2 |      payload2           |
//                     | tag3 |   payload3       |
//                            | tag 4 | hdr-data |
//
//    * payload shall always be encoded as CBOR byte-array.
//	  * hdr-data shall always be encoded as CBOR map.
//	  * tags are uint64 numbers that will either be prefixed
//      to payload or msg.
//
//       | tag1  | 0xff |
//
//    * if packet denotes a stream-end, payload will be
//      1-byte 0xff, and not encoded as byte-array.
//
//	  * tag1, will always be a opaque number falling within a
//      reserved tag-space called opaque-space.
//    * tag2, tag3 can be one of the values predefined by gofast.
//    * the final embedded tag, in this case tag4, shall always
//      be tagMsg (value 37).
//
// configurations:
//
//	"name"		   - give a name for the transport.
//	"buffersize"   - maximum size that a packet can needs.
//	"chansize"     - channel size to use for internal go-routines.
//	"batchsize"    - number of packets to batch before writting to socket.
//  "tags"		   - comma separated list of tags to apply, in specified order.
//	"opaque.start" - starting opaque range, inclusive.
//	"opaque.end"   - ending opaque range, inclusive.
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
import "reflect"
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
	streams  chan *Stream
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
		streams:  nil, // shall be initialized after setOpaqueRange() call
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

	// parse tag list, tags will be applied in the specified order.
	if tagcsv, ok := config["tags"].(string); ok {
		for _, tag := range strings.Split(tagcsv, ",") {
			if strings.Trim(tag, " \n\t\r") != "" {
				factory, ok := tag_factory[tag]
				if !ok {
					panic(fmt.Errorf("unknown tag %v", tag))
				}
				tagid, enc, dec := factory(t, config)
				t.tagenc[tagid], t.tagdec[tagid] = enc, dec
			}
		}
	}

	t.setOpaqueRange(uint64(opqstart), uint64(opqend)) // precise sequence

	t.SubscribeMessages(t.msghandler, &Whoami{}, &Ping{}, &Heartbeat{})
	log.Verbosef("%v pre-initialized ...\n", t.logprefix)

	go t.doTx()
	go t.syncRx()

	msg, err := t.Whoami()
	if err != nil {
		return nil, err
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
	factory := func(msg Message) func() interface{} {
		return func() interface{} {
			typeOfMsg := reflect.ValueOf(msg).Elem().Type()
			return reflect.New(typeOfMsg).Interface()
		}
	}
	msgstr := []string{}
	for _, msg := range msgs {
		id := msg.Id()
		t.messages[id] = msg
		t.msgpools[id] = &sync.Pool{New: factory(msg)}
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
	t.pktpool = nil
	close(t.killch)
	log.Infof("%v ... closed\n", t.logprefix)
	return t.conn.Close()
}

//---- maintenance APIs

// FlushPeriod to periodically flush batched packets.
func (t *Transport) FlushPeriod(ms time.Duration) {
	tick := time.Tick(ms)
	go func() {
		for {
			<-tick
			t.tx([]byte{} /*empty*/, true /*flush*/)
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
		}
	}()
}

// Liveat returns the timestamp of last heartbeat message received
// from peer.
func (t *Transport) Liveat() int64 {
	return atomic.LoadInt64(&t.aliveat)
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
	return t.post(msg)
}

// Request a response from peer.
func (t *Transport) Request(msg Message) (Message, error) {
	return t.request(msg)
}

// Request a bi-directional with peer.
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
	log.Debugf("%v local streams start %v end %v\n", t.logprefix, start, end)
	t.streams = make(chan *Stream, end-start+1) // inclusive
	for opaque := start; opaque <= end; opaque++ {
		t.streams <- t.newstream(uint64(opaque), false)
	}
}

type tagFactory func(*Transport, map[string]interface{}) (uint64, tagfn, tagfn)

var tag_factory = make(map[string]tagFactory)
