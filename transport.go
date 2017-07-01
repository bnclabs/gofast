package gofast

import "sync"
import "sync/atomic"
import "unsafe"
import "fmt"
import "strings"
import "net"
import "time"

import s "github.com/prataprc/gosettings"
import "github.com/prataprc/golog"

type tagfn func(in, out []byte) int
type fnTagFactory func(*Transport, s.Settings) (uint64, tagfn, tagfn)

var tagFactory = make(map[string]fnTagFactory)
var transports = unsafe.Pointer(&map[string]*Transporter{})

// RequestCallback handler called for an incoming post, or request,
// or stream message. On either side of the connection, RequestCallback
// initiates a new exchange - be it a Post or Request or Stream type.
// Applications should first register request-handler for expecting
// message-ids, and register a default handler to catch all other
// messages:
//
//   * Stream pointer will be nil if incoming message is a POST.
//   * Handler shall not block and must be as light-weight as possible
//   * If request is initiating a stream of messages from remote,
//   handler should return a stream-callback. StreamCallback will
//   dispatched for every new messages on this stream.
type RequestCallback func(*Stream, BinMessage) StreamCallback

// StreamCallback handler called for an incoming message on a stream,
// the boolean argument, if false, indicates whether remote has closed.
type StreamCallback func(BinMessage, bool)

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
	// statistics, keep this 8-byte aligned.
	nTx       uint64 // number of packets transmitted
	nFlushes  uint64 // number of times message-batches where flushed
	nTxbyte   uint64 // number of bytes transmitted on socket
	nTxpost   uint64 // number of post messages transmitted
	nTxreq    uint64 // number of request messages transmitted
	nTxresp   uint64 // number of response messages transmitted
	nTxstart  uint64 // number of start messages transmitted
	nTxstream uint64 // number of stream messages transmitted
	nTxfin    uint64 // number of finish messages transmitted
	nRx       uint64 // number of packets received
	nRxbyte   uint64 // number of bytes received from socket
	nRxpost   uint64 // number of post messages received
	nRxreq    uint64 // number of request messages received
	nRxresp   uint64 // number of response messages received
	nRxstart  uint64 // number of start messages received
	nRxstream uint64 // number of stream messages received
	nRxfin    uint64 // number of finish messages received
	nRxbeats  uint64 // number of heartbeats received
	nDropped  uint64 // number of dropped bytes
	nMdrops   uint64 // number of dropped messages

	// 0 no handshake
	// 1 oneway handshake
	// 2 bidirectional handshake
	xchngok int64

	// fields.
	name     string
	version  Version
	peerver  atomic.Value
	tagenc   map[uint64]tagfn   // tagid -> func
	tagdec   map[uint64]tagfn   // tagid -> func
	messages map[uint64]Message // msgid -> message
	handlers map[uint64]RequestCallback
	defaulth RequestCallback
	conn     Transporter
	aliveat  int64
	txch     chan *txproto
	rxch     chan rxpacket
	killch   chan struct{}

	// memory pools
	pStrms  chan *Stream // for locally initiated streams
	pTxcmd  chan *txproto
	pData   chan []byte
	pRxstrm *sync.Pool

	// settings
	settings   s.Settings
	buffersize uint64
	batchsize  uint64
	chansize   uint64
	logprefix  string
}

//---- transport initialization APIs

// NewTransport encapsulate a transport over this connection,
// one connection one transport.
func NewTransport(
	name string, conn Transporter, version Version,
	setts s.Settings) (*Transport, error) {

	if setts == nil {
		setts = DefaultSettings(1000, 5000)
	}

	buffersize := setts.Uint64("buffersize")
	opqstart := setts.Uint64("opaque.start")
	opqend := setts.Uint64("opaque.end")
	chansize := setts.Uint64("chansize")
	batchsize := setts.Uint64("batchsize")

	t := &Transport{
		name:    name,
		version: version,
		tagenc:  make(map[uint64]tagfn),
		tagdec:  make(map[uint64]tagfn),
		pStrms:  nil, // shall be initialized after setOpaqueRange() call
		pTxcmd:  nil, // shall be initialized after setOpaqueRange() call
		// TODO: avoid magic number
		pData:    make(chan []byte, 1000),
		messages: make(map[uint64]Message),
		handlers: make(map[uint64]RequestCallback),

		conn:   conn,
		txch:   make(chan *txproto, chansize+batchsize),
		rxch:   make(chan rxpacket, chansize),
		killch: make(chan struct{}),

		settings:   setts,
		batchsize:  batchsize,
		buffersize: buffersize,
		chansize:   chansize,
	}
	addtransport(name, t)

	laddr, raddr := conn.LocalAddr(), conn.RemoteAddr()
	t.logprefix = fmt.Sprintf("GFST[%v; %v<->%v]", name, laddr, raddr)
	t.pRxstrm = &sync.Pool{
		New: func() interface{} { return &Stream{} },
	}

	t.setOpaqueRange(uint64(opqstart), uint64(opqend))
	t.subscribeMessage(&whoamiMsg{}, t.msghandler)
	t.subscribeMessage(&pingMsg{}, t.msghandler)
	t.subscribeMessage(&heartbeatMsg{}, t.msghandler)

	// educate transport with configured tag decoders.
	tagcsv := setts.String("tags")
	for _, tag := range t.getTags(tagcsv, []string{}) {
		if factory, ok := tagFactory[tag]; ok {
			tagid, _, dec := factory(t, setts)
			t.tagdec[tagid] = dec
			continue
		}
		panic(fmt.Errorf("%v unknown tag %v", t.logprefix, tag))
	}
	log.Verbosef("%v pre-initialized ...\n", t.logprefix)

	go t.doTx()

	log.Infof("%v started ...\n", t.logprefix)
	return t, nil
}

// SubscribeMessage that shall be exchanged via this transport. Only
// subscribed messages can be exchanged. And for every incoming message
// with its ID equal to msg.ID(), handler will be dispatch.
//
// NOTE: handler shall not block and must be as light-weight as possible
func (t *Transport) SubscribeMessage(msg Message, handler RequestCallback) *Transport {
	id := msg.ID()
	if isReservedMsg(id) {
		panic(fmt.Errorf("%v message id %v reserved", t.logprefix, id))
	}
	return t.subscribeMessage(msg, handler)
}

// DefaultHandler register a default handler to handle all messages. If
// incoming message-id does not have a matching handler, default handler
// is dispatched, it is more like a catch-all for incoming messages.
//
// NOTE: handler shall not block and must be as light-weight as possible
func (t *Transport) DefaultHandler(handler RequestCallback) *Transport {
	t.defaulth = handler
	log.Verbosef("%v subscribed default handler\n", t.logprefix)
	return t
}

// Handshake with remote, shall be called after NewTransport(), before
// application messages are exchanged between nodes. Specifically, following
// information will be gathered from romote:
//   * Peer version, can later be queried via PeerVersion() API.
//   * Tags settings.
func (t *Transport) Handshake() *Transport {

	// now spawn the socket receiver, do this only after all messages
	// are subscribed.
	go t.syncRx() // shall spawn another go-routine doRx().

	wai, err := t.Whoami()
	if err != nil {
		panic(fmt.Errorf("%v Handshake(): %v", t.logprefix, err))
	}

	t.peerver.Store(wai.version)

	// parse tag list, tags shall be applied in the specified order.
	for _, tag := range t.getTags(wai.tags, []string{}) {
		if factory, ok := tagFactory[tag]; ok {
			tagid, enc, _ := factory(t, t.settings)
			t.tagenc[tagid] = enc
			continue
		}
		log.Warnf("%v remote ask for unknown tag: %v\n", t.logprefix, tag)
	}
	fmsg := "%v handshake completed with peer: %#v ...\n"
	log.Verbosef(fmsg, t.logprefix, wai)

	atomic.AddInt64(&t.xchngok, 1)
	for atomic.LoadInt64(&t.xchngok) < 2 { // wait till remote handshake
		time.Sleep(100 * time.Millisecond)
	}

	return t
}

// Close this transport, connection shall be closed as well.
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
	deltransport(t.name)
	log.Infof("%v ... closed\n", t.logprefix)
	// finally close the connection itself.
	return t.conn.Close()
}

// IsClosed return whether this transport is closed or not.
func (t *Transport) IsClosed() bool {
	select {
	case <-t.killch:
		return true
	default:
	}
	return false
}

//---- maintenance APIs

// Name returns the transport-name.
func (t *Transport) Name() string {
	return t.name
}

// Silentsince returns the timestamp of last heartbeat message received
// from peer. If ZERO, remote is not using the heart-beat mechanism.
func (t *Transport) Silentsince() time.Duration {
	if atomic.LoadInt64(&t.aliveat) == 0 {
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
	return t.peerver.Load().(Version)
}

// Stat shall return the stat counts for this transport.
// Refer gofast.Stat() api for more information.
func (t *Transport) Stat() map[string]uint64 {
	stats := map[string]uint64{
		"n_tx":       atomic.LoadUint64(&t.nTx),
		"n_flushes":  atomic.LoadUint64(&t.nFlushes),
		"n_txbyte":   atomic.LoadUint64(&t.nTxbyte),
		"n_txpost":   atomic.LoadUint64(&t.nTxpost),
		"n_txreq":    atomic.LoadUint64(&t.nTxreq),
		"n_txresp":   atomic.LoadUint64(&t.nTxresp),
		"n_txstart":  atomic.LoadUint64(&t.nTxstart),
		"n_txstream": atomic.LoadUint64(&t.nTxstream),
		"n_txfin":    atomic.LoadUint64(&t.nTxfin),
		"n_rx":       atomic.LoadUint64(&t.nRx),
		"n_rxbyte":   atomic.LoadUint64(&t.nRxbyte),
		"n_rxpost":   atomic.LoadUint64(&t.nRxpost),
		"n_rxreq":    atomic.LoadUint64(&t.nRxreq),
		"n_rxresp":   atomic.LoadUint64(&t.nRxresp),
		"n_rxstart":  atomic.LoadUint64(&t.nRxstart),
		"n_rxstream": atomic.LoadUint64(&t.nRxstream),
		"n_rxfin":    atomic.LoadUint64(&t.nRxfin),
		"n_rxbeats":  atomic.LoadUint64(&t.nRxbeats),
		"n_dropped":  atomic.LoadUint64(&t.nDropped),
		"n_mdrops":   atomic.LoadUint64(&t.nMdrops),
	}
	return stats
}

// Stats return consolidated counts of all transport objects.
// Refer gofast.Stat() api for more information.
func Stats() map[string]uint64 {
	stats := map[string]uint64{}

	op := atomic.LoadPointer(&transports)
	transm := (*map[string]*Transport)(op)
	for _, t := range *transm {
		s := t.Stat()
		for k, v := range s {
			if acc, ok := stats[k]; ok {
				stats[k] = acc + v
			} else {
				stats[k] = v
			}
		}
	}
	return stats
}

// Stat count for specified transport object.
//
// Available statistics:
//
// "n_tx"
//		number of messages transmitted.
//
// "n_flushes"
//		number of times message-batches where flushed.
//
// "n_txbyte"
//		number of bytes transmitted on socket.
//
// "n_txpost"
//		number of post messages transmitted.
//
// "n_txreq"
//		number of request messages transmitted.
//
// "n_txresp"
//		number of response messages transmitted.
//
// "n_txstart"
//		number of start messages transmitted, indicates the number of
//		streams started by this local node.
//
// "n_txstream"
//		number of stream messages transmitted.
//
// "n_txfin"
//		number of finish messages transmitted, indicates the number of
//		streams closed by this local node, should always match "n_txstart"
//		plus active streams.
//
// "n_rx"
//		number of packets received.
//
// "n_rxbyte"
//		number of bytes received from socket.
//
// "n_rxpost"
//		number of post messages received.
//
// "n_rxreq"
//		number of request messages received.
//
// "n_rxresp"
//		number of response messages received.
//
// "n_rxstart"
//		number of start messages received, indicates the number of
//		streams started by the remote node.
//
// "n_rxstream"
//		number of stream messages received.
//
// "n_rxfin"
//		number of finish messages received, indicates the number
//		of streams closed by the remote node, should always match
//		"n_rxstart" plus active streams.
//
// "n_rxbeats"
//		number of heartbeats received.
//
// "n_dropped"
//		bytes dropped.
//
// "n_mdrops"
//		messages dropped.
//
// Note that `n_dropped` and `n_mdrops` are counted because gofast
// supports either end to finish an ongoing stream of messages.
// It might be normal to see non-ZERO values.
func Stat(name string) map[string]uint64 {
	op := atomic.LoadPointer(&transports)
	transm := (*map[string]*Transport)(op)
	for transname, t := range *transm {
		if transname == name {
			return t.Stat()
		}
	}
	return nil
}

//---- transport APIs

// Whoami shall return remote's Whoami.
func (t *Transport) Whoami() (wai Whoami, err error) {
	req, resp := newWhoami(t), newWhoami(t)
	resp.transport = t
	if err = t.Request(req, true /*flush*/, resp); err != nil {
		return
	}
	return Whoami{whoamiMsg: *resp}, nil
}

// Ping pong with peer, returns the pong string.
func (t *Transport) Ping(echo string) (string, error) {
	req, resp := newPing(echo), &pingMsg{}
	if err := t.Request(req, true /*flush*/, resp); err != nil {
		return "", err
	}
	return resp.echo, nil
}

// Post request to peer.
func (t *Transport) Post(msg Message, flush bool) error {
	stream := t.getlocalstream(false /*tellrx*/, nil)
	defer t.putstream(stream.opaque, stream, false /*tellrx*/)

	n := t.post(msg, stream, stream.out)
	return t.txasync(stream.out[:n], flush)
}

// Request a response from peer. Caller is expected to pass reference to
// an expected response message, this also implies that every request can
// expect only one response type. This also have an added benefit of
// reducing the memory pressure on GC.
func (t *Transport) Request(msg Message, flush bool, resp Message) error {
	donech := make(chan struct{})
	stream := t.getlocalstream(true /*tellrx*/, func(bmsg BinMessage, ok bool) {
		if resp != nil {
			resp.Decode(bmsg.Data)
		}
		select {
		case <-donech:
		default:
			close(donech)
		}
	})
	defer t.putstream(stream.opaque, stream, true /*tellrx*/)

	n := t.request(msg, stream, stream.out)
	if err := t.tx(stream.out[:n], flush); err != nil {
		return err
	}
	<-donech
	stream.rxcallb = nil
	return nil
}

// Stream a bi-directional stream with peer.
func (t *Transport) Stream(
	msg Message, flush bool, rxcallb StreamCallback) (*Stream, error) {

	stream := t.getlocalstream(true /*tellrx*/, rxcallb)
	n := t.start(msg, stream, stream.out)
	if err := t.tx(stream.out[:n], false); err != nil {
		t.putstream(stream.opaque, stream, true /*tellrx*/)
		return nil, err
	}
	return stream, nil
}

//---- local APIs

func (t *Transport) setOpaqueRange(start, end uint64) {
	fmsg := "%v opaques should within [`%v`,`%v`], got  (%v,%v)"
	tagos, tagoe := TagOpaqueStart, TagOpaqueEnd
	if start < TagOpaqueStart {
		panic(fmt.Errorf(fmsg, t.logprefix, tagos, tagoe, start, end))
	} else if end > TagOpaqueEnd {
		panic(fmt.Errorf(fmsg, t.logprefix, tagos, tagoe, start, end))
	}
	fmsg = "%v local streams (%v,%v) pre-created\n"
	log.Debugf(fmsg, t.logprefix, start, end)

	t.pStrms = make(chan *Stream, end-start+1) // inclusive [start,end]
	for opaque := start; opaque <= end; opaque++ {
		if istagok(opaque) == false {
			continue
		}
		stream := &Stream{
			transport: t,
			remote:    false,
			opaque:    uint64(opaque),
			out:       make([]byte, t.buffersize),
			data:      make([]byte, t.buffersize),
			tagout:    make([]byte, t.buffersize),
		}
		t.pStrms <- stream
		fmsg := "%v ##%d(remote:%v) stream created ...\n"
		log.Verbosef(fmsg, t.logprefix, opaque, false)
	}

	t.pTxcmd = make(chan *txproto, end-start+1+uint64(t.batchsize))
	for i := 0; i < cap(t.pTxcmd); i++ {
		t.pTxcmd <- &txproto{packet: make([]byte, t.buffersize)}
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

func (t *Transport) subscribeMessage(m Message, h RequestCallback) *Transport {
	id := m.ID()
	t.messages[id] = m
	t.handlers[id] = h
	log.Verbosef("%v subscribed %v\n", t.logprefix, m)
	return t
}

func (t *Transport) requestCallback(s *Stream, msg BinMessage) StreamCallback {
	id := msg.ID
	if fn, ok := t.handlers[id]; ok && fn != nil {
		return fn(s, msg)
	} else if t.defaulth != nil {
		return t.defaulth(s, msg)
	}
	return nil
}

func (t *Transport) fromrxstrm() *Stream {
	stream := t.pRxstrm.Get().(*Stream)
	stream.transport, stream.rxcallb, stream.opaque = nil, nil, 0
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

func (t *Transport) fromtxpool() *txproto {
	arg := <-t.pTxcmd
	arg.flush, arg.async = false, false
	arg.n, arg.err, arg.respch = 0, nil, nil
	return arg
}

func (t *Transport) getdata(size int) (data []byte) {
	select {
	case data = <-t.pData:
		if cap(data) < size {
			data = make([]byte, size)
		}
	default:
		data = make([]byte, size)
	}
	return data[:size]
}

func (t *Transport) putdata(data []byte) {
	select {
	case t.pData <- data:
	default: // let GC collect the data
	}
}

// add a new transport.
func addtransport(name string, t *Transport) {
	for {
		op := atomic.LoadPointer(&transports)
		oldm := (*map[string]*Transport)(op)
		newm := map[string]*Transport{}
		for k, trans := range *oldm {
			if k == name {
				panic(fmt.Errorf("transport %v already created", name))
			}
			newm[k] = trans
		}
		newm[name] = t
		if atomic.CompareAndSwapPointer(&transports, op, unsafe.Pointer(&newm)) {
			return
		}
	}
}

// delete a transport.
func deltransport(name string) *Transport {
	for {
		op := atomic.LoadPointer(&transports)
		oldm := (*map[string]*Transport)(op)
		newm := map[string]*Transport{}
		for k, trans := range *oldm {
			newm[k] = trans
		}
		t, ok := newm[name]
		if !ok {
			panic(fmt.Errorf("transport %v not there", name))
		}
		delete(newm, name)
		if atomic.CompareAndSwapPointer(&transports, op, unsafe.Pointer(&newm)) {
			return t
		}
	}
}

func istagok(tag uint64) bool {
	if tag < 266 {
		return false
	} else if tag <= 1000 {
		return true
	} else if tag < 1004 {
		return false
	} else if tag <= 22097 {
		return true
	} else if tag == 22098 {
		return false
	} else if tag <= 55798 {
		return true
	} else if tag == 55799 {
		return false
	} else if tag <= 15309735 {
		return true
	} else {
		return false
	}
}
