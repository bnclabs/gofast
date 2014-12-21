package gofast

import "net"
import "sync"
import "time"
import "fmt"
import "errors"
import "io"
import "runtime/debug"

// ErrorClosed by client APIs when client instance is already
// closed, helpful when there is a race between API and Close()
var ErrorClosed = errors.New("gofast.clientClosed")

// Client connects with remote server via a single connection.
type Client struct {
	host string // remote address
	// transport
	muTransport sync.Mutex
	conn        net.Conn
	currOpaque  uint32
	raddr       string // actual port at remote side of connection.
	laddr       string // actual port at client side of connection.
	encoders    map[TransportFlag]Encoder
	zippers     map[TransportFlag]Compressor
	muxch       chan []interface{}
	demuxers    map[uint32]chan<- []interface{} // opaque -> receivers
	finch       chan bool
	// config params
	maxPayload     int           // for transport
	writeDeadline  time.Duration // timeout, in ms, while tx data to remote
	muxChanSize    int           // size of channel that mux requests
	streamChanSize int           // for each outstanding request
	log            Logger
	logPrefix      string
}

// NewClient returns a client instance with a single connection
// to host. Multiple go-routines can share this client and make
// API calls on this client.
func NewClient(
	host string, config map[string]interface{},
	log Logger) (c *Client, err error) {

	if log == nil {
		log = SystemLog("client-logger")
	}

	muxChanSize := config["muxChanSize"].(int)
	streamChanSize := config["streamChanSize"].(int)
	c = &Client{
		host: host,
		// transport
		currOpaque: uint32(1),
		encoders:   make(map[TransportFlag]Encoder),
		zippers:    make(map[TransportFlag]Compressor),
		muxch:      make(chan []interface{}, muxChanSize),
		demuxers:   make(map[uint32]chan<- []interface{}),
		finch:      make(chan bool),
		// config
		maxPayload:     config["maxPayload"].(int),
		writeDeadline:  time.Duration(config["writeDeadline"].(int)),
		muxChanSize:    muxChanSize,
		streamChanSize: streamChanSize,
		log:            log,
	}
	c.conn, err = net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	c.laddr, c.raddr = c.conn.LocalAddr().String(), c.conn.RemoteAddr().String()
	c.logPrefix = fmt.Sprintf("GFST[%v:%v<->%v]", host, c.laddr, c.raddr)
	return c, nil
}

// Start will start tx and rx routines for this client and connection.
// Shall be called only after all calls to SetEncoder() and
// SetZipper()
func (c *Client) Start() {
	go c.handleConnection(c.conn, c.muxch)
	go c.doReceive(c.conn)
}

// SetEncoder till set an encoder to encode and decode payload.
// Shall be called before starting the client.
func (c *Client) SetEncoder(typ TransportFlag, encoder Encoder) {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	c.encoders[typ] = encoder
}

// SetZipper will set a zipper type to compress and decompress payload.
// Shall be called before starting the client.
func (c *Client) SetZipper(typ TransportFlag, zipper Compressor) {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	c.zippers[typ] = zipper
}

func (c *Client) Close() {
	defer func() {
		if r := recover(); r != nil {
			c.log.Errorf("%v Close() crashed: %v\n", c.logPrefix, r)
			c.log.StackTrace(string(debug.Stack()))
		}
	}()

	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	if c.conn != nil {
		c.conn.Close()
		close(c.finch)
		c.conn = nil
		c.log.Infof("%v ... stopped\n", c.logPrefix)
	}
}

func (c *Client) Post(flags TransportFlag, payload interface{}) error {
	flags = flags.ClearStream().ClearEndStream().SetRequest()
	cmd := []interface{}{flags, OpaquePost, payload}
	return failsafeOpAsync(c.muxch, cmd, c.finch)
}

//------------------
// private functions
//------------------

func (c *Client) newRequest(opaque uint32) uint32 {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	if opaque == 0 {
		c.currOpaque++
		if c.currOpaque > 0x7FFFFFF { // TODO: no magic numbers
			c.currOpaque = 1
		}
		return c.currOpaque
	}
	return opaque
}

func (c *Client) setDemuxch(opaque uint32, ch chan<- []interface{}) {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	c.demuxers[opaque] = ch
}

func (c *Client) getDemuxch(opaque uint32) (chan<- []interface{}, bool) {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	ch, ok := c.demuxers[opaque]
	return ch, ok
}

func (c *Client) delDemuxch(opaque uint32) {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	delete(c.demuxers, opaque)
}

//--------------------
// Connection handling
//--------------------

// go-routine that waits for response and response stream messages.
func (c *Client) handleConnection(conn net.Conn, muxch <-chan []interface{}) {
	log, prefix := c.log, c.logPrefix
	tpkt := c.newTransport(conn)
	log.Debugf("%v handleConnection() ...\n", prefix)
	txmsg := "%v transmitting {%x,%x,%T}\n"

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("%v handleConnection() crashed: %v\n", prefix, r)
			StackTrace(string(debug.Stack()))
		}
		log.Debugf("%v handleConnection() exiting ...\n", prefix)
		go c.Close()
	}()

	timeoutMs := c.writeDeadline * time.Millisecond

loop:
	for {
		msg := <-muxch
		tflags, opaque := msg[0].(TransportFlag), msg[1].(uint32)
		payload := msg[2]
		conn.SetWriteDeadline(time.Now().Add(timeoutMs))
		if err := tpkt.Send(tflags, opaque, payload); err != nil {
			break loop
		}
		log.Tracef(txmsg, prefix, opaque, tflags, payload)
	}
}

// go-routine to push requests and request stream messages.
func (c *Client) doReceive(conn net.Conn) {
	log, prefix := c.log, c.logPrefix
	rpkt := c.newTransport(conn)
	log.Debugf("%v doReceive() ...\n", prefix)

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("%v doReceive() crashed: %v\n", prefix, r)
			StackTrace(string(debug.Stack()))
		}
		log.Debugf("%v doReceive() exiting ...\n", prefix)
		go c.Close()
	}()

loop:
	for {
		// Note: receive on connection shall work without timeout.
		// will exit only when remote fails or server is closed.
		mtype, flags, opaque, payload, err := rpkt.Receive()
		if err != nil {
			if err != io.EOF {
				log.Errorf("%v remote exited: %v\n", prefix, err)
			}
			break loop
		}
		if respch, ok := c.getDemuxch(opaque); ok {
			respch <- []interface{}{mtype, flags, opaque, payload}
		} else { // ignore reponses if opaque not found.
		}
	}
}

func (c *Client) newTransport(conn net.Conn) *TransportPacket {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	pkt := NewTransportPacket(conn, c.maxPayload, c.log)
	for flag, encoder := range c.encoders {
		pkt.SetEncoder(flag.GetEncoding(), encoder)
	}
	for flag, zipper := range c.zippers {
		pkt.SetZipper(flag.GetCompression(), zipper)
	}
	return pkt
}

//----------------
// local functions
//----------------

// failsafeOp can be used by gen-server implementors to avoid infinitely
// blocked API calls.
func failsafeOp(
	muxch, respch chan []interface{}, cmd []interface{},
	finch chan bool) ([]interface{}, error) {

	select {
	case muxch <- cmd:
		if respch != nil {
			select {
			case resp := <-respch:
				return resp, nil
			case <-finch:
				return nil, ErrorClosed
			}
		}
	case <-finch:
		return nil, ErrorClosed
	}
	return nil, nil
}

// failsafeOpAsync is same as FailsafeOp that can be used for
// asynchronous operation, that is, caller does not wait for response.
func failsafeOpAsync(
	muxch chan []interface{}, cmd []interface{}, finch chan bool) error {

	select {
	case muxch <- cmd:
	case <-finch:
		return ErrorClosed
	}
	return nil
}
