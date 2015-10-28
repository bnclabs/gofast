//  Copyright (c) 2014 Couchbase, Inc.

// Package gofast implements a high performance symmetric protocol for on the
// wire data transport.
//
// opaque-space, is range of uint64 values reserved for tagging packets. They
// shall be supplied via configuration while instantiating the transport.
//
// messages, are golang objects implementing the Message{} interface. Message
// objects need to be subscribed with transport before they are exchanged over
// the transport. It is also expected that distributed systems must
// pre-define messages and their Ids.
//
// transport instantiation steps:
//
//		t := NewTransport(conn, &ver, nil, config)
//		t.SubscribeMessage(&msg1, handler1) // subscribe message
//		t.SubscribeMessage(&msg2, handler2) // subscribe another message
//		t.Handshake()
//		t.FlushPeriod(tm)				  // optional
//		t.SendHeartbeat(tm)				  // optional
//
// incoming messages are created from *sync.Pool, handlers (like handler1 and
// handler2 in the above eg.) can return the message back to the pool using
// the Free() method on the transport.
package gofast

import "sync"
import "sync/atomic"
import "fmt"
import "strings"
import "net"
import "time"

type tagfn func(in, out []byte) int

// RequestCallback handler called for an incoming post, request or stream;
// to be supplied by application before using the transport.
type RequestCallback func(*Stream, Message) chan Message

// Transporter interface to send and receive packets, connection object
// shall implement this interface.
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
	tagenc   map[uint64]tagfn   // tagid -> func
	tagdec   map[uint64]tagfn   // tagid -> func
	messages map[uint64]Message // msgid -> message
	handlers map[uint64]RequestCallback
	conn     Transporter
	aliveat  int64
	txch     chan *txproto
	rxch     chan rxpacket
	killch   chan bool
	// 0 no handshake
	// 1 oneway handshake
	// 2 bidirectional handshake
	xchngok int64

	// mempools
	strmpool chan *Stream // for locally initiated streams
	p_rqrch  chan chan Message
	p_txcmd  *sync.Pool
	p_txacmd *sync.Pool
	p_rxstrm *sync.Pool
	msgpools map[uint64]*sync.Pool

	// configuration
	config     map[string]interface{}
	buffersize int
	logprefix  string

	// statistics
	n_tx       uint64 // number of packets transmitted
	n_flushes  uint64 // number of times message-batches where flushed
	n_txbyte   uint64 // number of bytes transmitted on socket
	n_txpost   uint64 // number of post messages transmitted
	n_txreq    uint64 // number of request messages transmitted
	n_txresp   uint64 // number of response messages transmitted
	n_txstart  uint64 // number of start messages transmitted
	n_txstream uint64 // number of stream messages transmitted
	n_txfin    uint64 // number of finish messages transmitted
	n_rx       uint64 // number of packets received
	n_rxbyte   uint64 // number of bytes received from socket
	n_rxpost   uint64 // number of post messages received
	n_rxreq    uint64 // number of request messages received
	n_rxresp   uint64 // number of response messages received
	n_rxstart  uint64 // number of start messages received
	n_rxstream uint64 // number of stream messages received
	n_rxfin    uint64 // number of finish messages received
	n_rxbeats  uint64 // number of heartbeats received
	n_dropped  uint64 // number of dropped bytes
	n_mdrops   uint64 // number of dropped messages
}

//---- transport initialization APIs

// NewTransport encapsulate a transport over this connection,
// one connection one transport.
func NewTransport(conn Transporter, version Version, logg Logger, config map[string]interface{}) (*Transport, error) {
	name := config["name"].(string)
	buffersize := config["buffersize"].(int)
	opqstart := config["opaque.start"].(int)
	opqend := config["opaque.end"].(int)
	chansize := config["chansize"].(int)
	batchsize := config["batchsize"].(int)

	t := &Transport{
		name:     name,
		version:  version,
		tagenc:   make(map[uint64]tagfn),
		tagdec:   make(map[uint64]tagfn),
		strmpool: nil, // shall be initialized after setOpaqueRange() call
		p_rqrch:  nil, // shall be initialized after setOpaqueRange() call
		messages: make(map[uint64]Message),
		handlers: make(map[uint64]RequestCallback),

		conn:   conn,
		txch:   make(chan *txproto, chansize+batchsize),
		rxch:   make(chan rxpacket, chansize),
		killch: make(chan bool),

		msgpools: make(map[uint64]*sync.Pool),

		config:     config,
		buffersize: buffersize,
	}

	setLogger(logg, t.config)

	laddr, raddr := conn.LocalAddr(), conn.RemoteAddr()
	t.logprefix = fmt.Sprintf("GFST[%v; %v<->%v]", name, laddr, raddr)
	t.p_txcmd = &sync.Pool{
		New: func() interface{} { return &txproto{} },
	}
	t.p_txacmd = &sync.Pool{ // async commands
		New: func() interface{} { return &txproto{} },
	}
	t.p_rxstrm = &sync.Pool{
		New: func() interface{} { return &Stream{} },
	}

	t.setOpaqueRange(uint64(opqstart), uint64(opqend))
	t.subscribeMessage(&Whoami{}, t.msghandler)
	t.subscribeMessage(&Ping{}, t.msghandler)
	t.subscribeMessage(&Heartbeat{}, t.msghandler)

	// educate tranport with configured tag decoders.
	tagcsv, _ := config["tags"]
	for _, tag := range t.getTags(tagcsv.(string), []string{}) {
		if factory, ok := tag_factory[tag]; ok {
			tagid, _, dec := factory(t, config)
			t.tagdec[tagid] = dec
			continue
		}
		panic(fmt.Errorf("unknown tag %v", tag))
	}
	log.Verbosef("%v pre-initialized ...\n", t.logprefix)

	go t.doTx()
	go t.syncRx() // shall spawn another go-routine doRx().

	log.Infof("%v started ...\n", t.logprefix)
	return t, nil
}

// Handshake with remote, shall be called after NewTransport().
func (t *Transport) Handshake() *Transport {
	msg, err := t.Whoami()
	if err != nil {
		panic(err)
	}

	t.peerver = msg.version // TODO: should be atomic ?
	// parse tag list, tags shall be applied in the specified order.
	for _, tag := range t.getTags(msg.tags, []string{}) {
		if factory, ok := tag_factory[tag]; ok {
			tagid, enc, _ := factory(t, t.config)
			t.tagenc[tagid] = enc
			continue
		}
		log.Warnf("%v remote ask for unknown tag: %v", t.logprefix, tag)
	}
	fmsg := "%v handshake completed with peer: %#v ...\n"
	log.Verbosef(fmsg, t.logprefix, msg)
	atomic.AddInt64(&t.xchngok, 1)
	for atomic.LoadInt64(&t.xchngok) < 2 { // wait till remote handshake
		time.Sleep(100 * time.Millisecond)
	}
	return t
}

// SubscribeMessage that shall be exchanged via this transport. Only
// subscribed messages can be exchanged.
func (t *Transport) SubscribeMessage(msg Message, handler RequestCallback) *Transport {
	id := msg.Id()
	if isReservedMsg(id) {
		panic(fmt.Errorf("message id %v reserved", id))
	}
	return t.subscribeMessage(msg, handler)
}

// Close this tranport, connection shall be closed as well.
func (t *Transport) Close() error {
	defer func() {
		if r := recover(); r != nil {
			fmsg := "%v transport.Close() recovered: %v\n"
			log.Infof(fmsg, t.logprefix, r)
		}
	}()
	// closing kill-channel should accomplish the following,
	// a. prevent any more transmission on the connection.
	// b. close all active streams.
	close(t.killch)
	log.Infof("%v ... closed\n", t.logprefix)
	// finally close the connection itself.
	return t.conn.Close()
}

//---- maintenance APIs

// Name returns the transport-name.
func (t *Transport) Name() string {
	return t.name
}

// FlushPeriod to periodically flush batched packets.
func (t *Transport) FlushPeriod(ms time.Duration) {
	tick := time.Tick(ms)
	go func() {
		for {
			<-tick
			if t.tx([]byte{} /*empty*/, true /*flush*/) != nil {
				return
			}
			//log.Debugf("%v flushed ... \n", t.logprefix)
		}
	}()
}

// SendHeartbeat to periodically send keep-alive message.
func (t *Transport) SendHeartbeat(ms time.Duration) {
	count, tick := uint64(0), time.Tick(ms)
	go func() {
		for {
			<-tick
			msg := NewHeartbeat(count)
			if t.Post(msg, true /*flush*/) != nil {
				return
			}
			count++
			//log.Debugf("%v posted heartbeat %v\n", t.logprefix, count)
		}
	}()
}

// Silentsince returns the timestamp of last heartbeat message received
// from peer.
func (t *Transport) Silentsince() time.Duration {
	if t.aliveat == 0 {
		log.Warnf("%v heartbeat not initialized\n", t.logprefix)
		return time.Duration(0)
	}
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

// Free shall return the message back to the pool, should be
// called on messages received via RequestCallback callback and
// via Stream.Rxch.
func (t *Transport) Free(msg Message) {
	t.msgpools[msg.Id()].Put(msg)
}

// Counts shall return the stat counts for this transport.
func (t *Transport) Counts() map[string]uint64 {
	stats := map[string]uint64{
		"n_tx":       atomic.LoadUint64(&t.n_tx),
		"n_flushes":  atomic.LoadUint64(&t.n_flushes),
		"n_txbyte":   atomic.LoadUint64(&t.n_txbyte),
		"n_txpost":   atomic.LoadUint64(&t.n_txpost),
		"n_txreq":    atomic.LoadUint64(&t.n_txreq),
		"n_txresp":   atomic.LoadUint64(&t.n_txresp),
		"n_txstart":  atomic.LoadUint64(&t.n_txstart),
		"n_txstream": atomic.LoadUint64(&t.n_txstream),
		"n_txfin":    atomic.LoadUint64(&t.n_txfin),
		"n_rx":       atomic.LoadUint64(&t.n_rx),
		"n_rxbyte":   atomic.LoadUint64(&t.n_rxbyte),
		"n_rxpost":   atomic.LoadUint64(&t.n_rxpost),
		"n_rxreq":    atomic.LoadUint64(&t.n_rxreq),
		"n_rxresp":   atomic.LoadUint64(&t.n_rxresp),
		"n_rxstart":  atomic.LoadUint64(&t.n_rxstart),
		"n_rxstream": atomic.LoadUint64(&t.n_rxstream),
		"n_rxfin":    atomic.LoadUint64(&t.n_rxfin),
		"n_rxbeats":  atomic.LoadUint64(&t.n_rxbeats),
		"n_dropped":  atomic.LoadUint64(&t.n_dropped),
		"n_mdrops":   atomic.LoadUint64(&t.n_mdrops),
	}
	return stats
}

//---- transport APIs

// Whoami shall return remote transport's information.
func (t *Transport) Whoami() (*Whoami, error) {
	msg := NewWhoami(t)
	resp, err := t.Request(msg, true /*flush*/)
	if err != nil {
		return nil, err
	}
	return resp.(*Whoami), nil
}

// Ping pong with peer.
func (t *Transport) Ping(echo string) (*Ping, error) {
	msg := NewPing(echo)
	resp, err := t.Request(msg, true /*flush*/)
	if err != nil {
		return nil, err
	}
	return resp.(*Ping), nil
}

// Post request to peer.
func (t *Transport) Post(msg Message, flush bool) error {
	stream := t.getstream(nil)
	defer t.putstream(stream.opaque, stream, true /*tellrx*/)

	n := t.post(msg, stream, stream.out)
	return t.txasync(stream.out[:n], flush)
}

// Request a response from peer.
func (t *Transport) Request(msg Message, flush bool) (resp Message, err error) {
	respch := <-t.p_rqrch
	stream := t.getstream(respch)
	defer t.putstream(stream.opaque, stream, true /*tellrx*/)
	defer func() { t.p_rqrch <- respch }()

	var ok bool
	n := t.request(msg, stream, stream.out)
	if err = t.tx(stream.out[:n], flush); err == nil {
		resp, ok = <-stream.Rxch
		if !ok {
			err = fmt.Errorf("stream closed")
		}
	}
	stream.Rxch = nil // so that p_rqrch channels are not closed !!
	return
}

// Request a bi-directional stream with peer.
func (t *Transport) Stream(msg Message, flush bool, ch chan Message) (*Stream, error) {
	stream := t.getstream(ch)
	n := t.start(msg, stream, stream.out)
	if err := t.tx(stream.out[:n], false); err != nil {
		t.putstream(stream.opaque, stream, true /*tellrx*/)
		return nil, err
	}
	return stream, nil
}

//---- local APIs

func (t *Transport) setOpaqueRange(start, end uint64) {
	fmsg := "opaques must start `%v` and shall not exceed `%v`(%v,%v)"
	err := fmt.Errorf(fmsg, tagOpaqueStart, tagOpaqueEnd, start, end)
	if start < tagOpaqueStart {
		panic(err)
	} else if end > tagOpaqueEnd {
		panic(err)
	}
	log.Debugf("%v local streams (%v,%v) pre-created\n", t.logprefix, start, end)
	t.strmpool = make(chan *Stream, end-start+1) // inclusive
	t.p_rqrch = make(chan chan Message, end-start+1)
	for opaque := start; opaque <= end; opaque++ {
		stream := &Stream{
			transport: t,
			remote:    false,
			opaque:    uint64(opaque),
			Rxch:      nil,
			out:       make([]byte, t.buffersize),
			data:      make([]byte, t.buffersize),
			tagout:    make([]byte, t.buffersize),
		}
		t.strmpool <- stream
		t.p_rqrch <- make(chan Message, 1)
		fmsg := "%v ##%d(remote:%v) stream created ...\n"
		log.Verbosef(fmsg, t.logprefix, opaque, false)
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

func (t *Transport) subscribeMessage(
	msg Message, handler RequestCallback) *Transport {

	id := msg.Id()
	t.messages[id] = msg
	t.msgpools[id] = &sync.Pool{New: msgfactory(msg)}
	t.handlers[id] = handler

	log.Verbosef("%v subscribed %v\n", t.logprefix, msg)
	return t
}

func (t *Transport) requestCallback(stream *Stream, msg Message) chan Message {
	id := msg.Id()
	if fn := t.handlers[id]; fn != nil {
		return fn(stream, msg)
	}
	log.Warnf("request-callback registered nil for msg:(%v,%T)\n", id, msg)
	return nil
}

func (t *Transport) fromrxstrm() *Stream {
	stream := t.p_rxstrm.Get().(*Stream)
	stream.transport, stream.Rxch, stream.opaque = nil, nil, 0
	stream.remote = false
	if stream.out == nil {
		stream.out = make([]byte, t.buffersize)
	}
	if stream.data == nil {
		stream.data = make([]byte, t.buffersize)
	}
	if stream.tagout == nil {
		stream.tagout = make([]byte, t.buffersize)
	}
	return stream
}

func (t *Transport) fromtxpool(async bool, pool *sync.Pool) (arg *txproto) {
	if async {
		arg = pool.Get().(*txproto)
		arg.flush, arg.async = false, false
		arg.n, arg.err, arg.respch = 0, nil, nil
		if arg.packet == nil {
			arg.packet = make([]byte, t.buffersize)
		}
	} else {
		arg = pool.Get().(*txproto)
		arg.packet, arg.flush, arg.async = nil, false, false
		arg.n, arg.err, arg.respch = 0, nil, nil
	}
	return arg
}

type tagFactory func(*Transport, map[string]interface{}) (uint64, tagfn, tagfn)

var tag_factory = make(map[string]tagFactory)
