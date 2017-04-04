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
type tagFactory func(*Transport, s.Settings) (uint64, tagfn, tagfn)

var tag_factory = make(map[string]tagFactory)
var transports = unsafe.Pointer(&map[string]*Transporter{})

// StreamCallback handler called for an incoming message on a stream,
// the boolean argument indicates whether remote has closed the stream.
type StreamCallback func(BinMessage, bool)

// RequestCallback handler called for an incoming post, request or stream;
// to be supplied by application before using the transport.
type RequestCallback func(*Stream, BinMessage) StreamCallback

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
	p_strms  chan *Stream // for locally initiated streams
	p_txcmd  chan *txproto
	p_data   chan []byte
	p_rxstrm *sync.Pool

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
		p_strms: nil, // shall be initialized after setOpaqueRange() call
		p_txcmd: nil, // shall be initialized after setOpaqueRange() call
		// TODO: avoid magic number
		p_data:   make(chan []byte, 1000),
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
	t.p_rxstrm = &sync.Pool{
		New: func() interface{} { return &Stream{} },
	}

	t.setOpaqueRange(uint64(opqstart), uint64(opqend))
	t.subscribeMessage(&whoamiMsg{}, t.msghandler)
	t.subscribeMessage(&pingMsg{}, t.msghandler)
	t.subscribeMessage(&heartbeatMsg{}, t.msghandler)

	// educate transport with configured tag decoders.
	tagcsv := setts.String("tags")
	for _, tag := range t.getTags(tagcsv, []string{}) {
		if factory, ok := tag_factory[tag]; ok {
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
// subscribed messages can be exchanged.
func (t *Transport) SubscribeMessage(msg Message, handler RequestCallback) *Transport {
	id := msg.ID()
	if isReservedMsg(id) {
		panic(fmt.Errorf("%v message id %v reserved", t.logprefix, id))
	}
	return t.subscribeMessage(msg, handler)
}

func (t *Transport) DefaultHandler(handler RequestCallback) *Transport {
	t.defaulth = handler
	log.Verbosef("%v subscribed default handler\n", t.logprefix)
	return t
}

// Handshake with remote, shall be called after NewTransport().
func (t *Transport) Handshake() *Transport {

	// now spawn the socket receiver, do this only after all messages
	// are subscribed.
	go t.syncRx() // shall spawn another go-routine doRx().

	msg, err := t.Whoami()
	if err != nil {
		panic(fmt.Errorf("%v Handshake(): %v", t.logprefix, err))
	}

	wai := msg.(*whoamiMsg)
	t.peerver.Store(wai.version)

	// parse tag list, tags shall be applied in the specified order.
	for _, tag := range t.getTags(wai.tags, []string{}) {
		if factory, ok := tag_factory[tag]; ok {
			tagid, enc, _ := factory(t, t.settings)
			t.tagenc[tagid] = enc
			continue
		}
		log.Warnf("%v remote ask for unknown tag: %v\n", t.logprefix, tag)
	}
	fmsg := "%v handshake completed with peer: %#v ...\n"
	log.Verbosef(fmsg, t.logprefix, msg)

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
		return false
	}
	panic("unreachable code")
}

//---- maintenance APIs

// Name returns the transport-name.
func (t *Transport) Name() string {
	return t.name
}

// Silentsince returns the timestamp of last heartbeat message received
// from peer.
func (t *Transport) Silentsince() time.Duration {
	if atomic.LoadInt64(&t.aliveat) == 0 {
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
	return t.peerver.Load().(Version)
}

// Stat shall return the stat counts for this transport. Refer gofast.Stats()
// api for more information.
func (t *Transport) Stat() map[string]uint64 {
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

// Stats return consolidated counts of all transport objects.
// Refer gofast.Stats() api for more information.
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
// 		number of messages transmitted.
//
// "n_flushes"
//		number of times message-batches where flushed.
//
// "n_txbyte"
//     	number of bytes transmitted on socket.
//
// "n_txpost"
//     	number of post messages transmitted.
//
// "n_txreq"
//     	number of request messages transmitted.
//
// "n_txresp"
//     	number of response messages transmitted.
//
// "n_txstart"
//     	number of start messages transmitted, indicates the number of
//     	streams started by this local node.
//
// "n_txstream"
//     	number of stream messages transmitted.
//
// "n_txfin"
//     	number of finish messages transmitted, indicates the number of
//     	streams closed by this local node, should always match "n_txstart".
//
// "n_rx"
//     	number of packets received.
//
// "n_rxbyte"
//     	number of bytes received from socket.
//
// "n_rxpost"
//     	number of post messages received.
//
// "n_rxreq"
//     	number of request messages received.
//
// "n_rxresp"
//     	number of response messages received.
//
// "n_rxstart"
//     	number of start messages received, indicates the number of
//     	streams started by the remote node.
//
// "n_rxstream"
//     	number of stream messages received.
//
// "n_rxfin"
//     	number of finish messages received, indicates the number
//     	of streams closed by the remote node, should always match
//     	"n_rxstart"
//
// "n_rxbeats"
//     	number of heartbeats received.
//
// "n_dropped"
//     	bytes dropped.
//
// "n_mdrops"
//     	messages dropped.
//
// Note that `n_dropped` and `n_mdrops` are counted because gofast
// supports either end to finish an ongoing stream of messages.
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

// Whoami shall return remote transport's information.
func (t *Transport) Whoami() (Message, error) {
	req, resp := newWhoami(t), newWhoami(t)
	resp.transport = t
	if err := t.Request(req, true /*flush*/, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Ping pong with peer.
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

// Request a response from peer.
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

// Request a bi-directional stream with peer.
func (t *Transport) Stream(msg Message, flush bool, rxcallb StreamCallback) (*Stream, error) {
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
	tagos, tagoe := tagOpaqueStart, tagOpaqueEnd
	if start < tagOpaqueStart {
		panic(fmt.Errorf(fmsg, t.logprefix, tagos, tagoe, start, end))
	} else if end > tagOpaqueEnd {
		panic(fmt.Errorf(fmsg, t.logprefix, tagos, tagoe, start, end))
	}
	fmsg = "%v local streams (%v,%v) pre-created\n"
	log.Debugf(fmsg, t.logprefix, start, end)

	t.p_strms = make(chan *Stream, end-start+1) // inclusive [start,end]
	for opaque := start; opaque <= end; opaque++ {
		stream := &Stream{
			transport: t,
			remote:    false,
			opaque:    uint64(opaque),
			out:       make([]byte, t.buffersize),
			data:      make([]byte, t.buffersize),
			tagout:    make([]byte, t.buffersize),
		}
		t.p_strms <- stream
		fmsg := "%v ##%d(remote:%v) stream created ...\n"
		log.Verbosef(fmsg, t.logprefix, opaque, false)
	}

	t.p_txcmd = make(chan *txproto, end-start+1+uint64(t.batchsize))
	for i := 0; i < cap(t.p_txcmd); i++ {
		t.p_txcmd <- &txproto{packet: make([]byte, t.buffersize)}
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
	stream := t.p_rxstrm.Get().(*Stream)
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
	arg := <-t.p_txcmd
	arg.flush, arg.async = false, false
	arg.n, arg.err, arg.respch = 0, nil, nil
	return arg
}

func (t *Transport) getdata(size int) (data []byte) {
	select {
	case data = <-t.p_data:
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
	case t.p_data <- data:
	default: // let GC collect the data
	}
}

// add a new trasnport.
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
