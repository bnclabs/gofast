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

var ErrorDuplicateRequest = errors.New("gofast.duplicateRequest")

var ErrorBadRequest = errors.New("gofast.badRequest")

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
	requests    map[uint32]*clientRequest
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
		requests:   make(map[uint32]*clientRequest),
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

// SetEncoder till set encoder handler to encode and decode payload.
// Shall be called before starting the client.
func (c *Client) SetEncoder(typ TransportFlag, encoder Encoder) {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	c.encoders[typ] = encoder
}

// SetZipper will set zipper handler to compress and decompress
// payload. Shall be called before starting the client.
func (c *Client) SetZipper(typ TransportFlag, zipper Compressor) {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	c.zippers[typ] = zipper
}

// Close this client
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

// Post payload to remote server, don't wait for response.
func (c *Client) Post(flags TransportFlag, payload interface{}) error {
	flags = flags.ClearStream().ClearEndStream().SetRequest()
	cmd := []interface{}{flags, OpaquePost, payload}
	return failsafeOpAsync(c.muxch, cmd, c.finch)
}

// Request server and wait for a single response from server
// and return response from server.
func (c *Client) Request(
	flags TransportFlag,
	payload interface{}) (response interface{}, err error) {

	return c.RequestWith(0, flags, payload)
}

// Request server and wait for a single response from server
// and return response from server.
func (c *Client) RequestWith(
	opaque uint32,
	flags TransportFlag,
	request interface{}) (response interface{}, err error) {

	flags = flags.SetRequest().SetStream().SetEndStream()
	rflags := TransportFlag(flags.GetEncoding())
	rflags = rflags.SetCompression(flags.GetCompression())
	respch := make(chan []interface{}, 1)
	cmd := []interface{}{rflags, opaque, request, respch}
	resp, err := failsafeOp(c.muxch, respch, cmd, c.finch)
	if err := opError(err, resp, 1); err != nil {
		return nil, err
	}
	return resp[0], nil
}

//------------------
// private functions
//------------------

type clientRequest struct {
	flags  TransportFlag
	opaque uint32
	respch chan<- []interface{}
}

func (c *Client) addRequest(
	flags TransportFlag,
	opaque uint32,
	respch chan<- []interface{}) *clientRequest {

	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	if opaque == 0 {
		c.currOpaque++
		if c.currOpaque > 0x7FFFFFF { // TODO: no magic numbers
			c.currOpaque = 1
		}
		opaque = c.currOpaque
	}
	cr := &clientRequest{flags, opaque, respch}
	c.setRequest(opaque, cr)
	return cr
}

func (c *Client) setRequest(opaque uint32, cr *clientRequest) {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	c.requests[opaque] = cr
}

func (c *Client) getRequest(opaque uint32) (*clientRequest, bool) {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	cr, ok := c.requests[opaque]
	return cr, ok
}

func (c *Client) delRequest(opaque uint32) {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	delete(c.requests, opaque)
}

//--------------------
// Connection handling
//--------------------

// go-routine that waits for response and response stream messages.
func (c *Client) handleConnection(conn net.Conn, muxch <-chan []interface{}) {
	log, prefix := c.log, c.logPrefix
	tpkt := c.newTransport(conn)
	log.Debugf("%v handleConnection() ...\n", prefix)
	txmsg := "%v transmitting %v {%x,%x,%T}\n"

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
		err := error(nil)
		msg := <-muxch
		flags, opaque := msg[0].(TransportFlag), msg[1].(uint32)
		payload, respch := msg[2], msg[3].(chan<- []interface{})
		f_r, f_s := flags.IsRequest(), flags.IsStream()
		f_e := flags.IsEndStream()
		_, known := c.requests[opaque]
		if f_r { // new request
			if f_s == false && f_e == false && known == false {
				log.Tracef(txmsg, prefix, "POST", flags, opaque, payload)

			} else if f_s && f_e && known == false {
				c.addRequest(flags, opaque, respch)
				log.Tracef(txmsg, prefix, "RQST", flags, opaque, payload)

			} else if known {
				err = ErrorDuplicateRequest
				log.Errorf(txmsg, prefix, "DUPE", flags, opaque, payload)
				respch <- []interface{}{err}

			} else {
				err = ErrorBadRequest
				log.Errorf(txmsg, prefix, "BADR", flags, opaque, payload)
				respch <- []interface{}{err}
			}

		} else { // streaming request
		}
		if err == nil {
			conn.SetWriteDeadline(time.Now().Add(timeoutMs))
			if err := tpkt.Send(flags, opaque, payload); err != nil {
				break loop
			}
		}
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
		mtype, _, opaque, payload, err := rpkt.Receive()
		if err != nil {
			if err != io.EOF {
				log.Errorf("%v remote exited: %v\n", prefix, err)
			}
			break loop
		}
		if cr, ok := c.getRequest(opaque); ok {
			cr.respch <- []interface{}{mtype, payload}

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
