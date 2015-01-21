package gofast

import "net"
import "sync"
import "time"
import "fmt"
import "io"
import "runtime/debug"

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
	requests    map[uint32]*clientRequest // opaque -> in-flight-request
	muxch       chan []interface{}
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
	host string,
	config map[string]interface{},
	log Logger) (c *Client, err error) {

	if log == nil {
		log = SystemLog("client-logger")
	}

	muxChanSize := config["muxChanSize"].(int)
	c = &Client{
		host: host,
		// transport
		currOpaque: uint32(0),
		encoders:   make(map[TransportFlag]Encoder),
		zippers:    make(map[TransportFlag]Compressor),
		requests:   make(map[uint32]*clientRequest),
		muxch:      make(chan []interface{}, muxChanSize),
		finch:      make(chan bool),
		// config
		maxPayload:     config["maxPayload"].(int),
		writeDeadline:  time.Duration(config["writeDeadline"].(int)),
		muxChanSize:    muxChanSize,
		streamChanSize: config["streamChanSize"].(int),
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

// Start will start tx and rx routines for this client
// and its connection, shall be called only after making
// calls to SetEncoder() and SetZipper()
func (c *Client) Start() {
	go c.handleConnection(c.conn, c.muxch)
	go c.doReceive(c.conn)
}

// SetEncoder till set encoder handler to encode and decode payload.
// Shall be called before starting the client.
func (c *Client) SetEncoder(typ TransportFlag, encoder Encoder) *Client {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	c.encoders[typ] = encoder
	return c
}

// SetZipper will set zipper handler to compress and decompress
// payload. Shall be called before starting the client.
func (c *Client) SetZipper(typ TransportFlag, zipper Compressor) *Client {
	c.muTransport.Lock()
	defer c.muTransport.Unlock()
	c.zippers[typ] = zipper
	return c
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
// Asynchronous call.
//
// - `flags` can be set with a supported encoding and
//    compression type.
func (c *Client) Post(
	flags TransportFlag, mtype uint16, payload interface{}) error {

	var respch chan []interface{}
	flags = flags.ClearStream().ClearEndStream().SetRequest()
	cmd := []interface{}{mtype, flags, NoOpaque, payload, respch}
	return failsafeOpAsync(c.muxch, cmd, c.finch)
}

// Request server and wait for a single response from server
// and return response from server.
// - `flags` refer to transport spec.
// - `payload` is actual request.
// - `mtype` is 16-bit value that can be used to interpret payload.
func (c *Client) Request(
	flags TransportFlag,
	mtype uint16,
	request interface{}) (response interface{}, err error) {

	flags = flags.SetRequest().SetStream().SetEndStream()
	respch := make(chan []interface{}, 1)
	cmd := []interface{}{mtype, flags, NoOpaque, request, respch}
	resp, err := failsafeOp(c.muxch, respch, cmd, c.finch)
	if err := opError(err, resp, 1); err != nil {
		return nil, err
	}
	return resp[0], nil
}

//-----------------
// request handling
//-----------------

type clientRequest struct {
	flags  TransportFlag
	opaque uint32
	rqst   bool
	strm   bool
	respch chan<- []interface{}
}

func (c *Client) addRequest(
	flags TransportFlag,
	respch chan<- []interface{}) (*clientRequest, uint32) {

	// Note: Request streams cannot switch encoding and
	// compression after the request is initiated.
	c.currOpaque++
	if c.currOpaque > 0x7FFFFFFF { // TODO: no magic numbers
		c.currOpaque = 1
	}
	tflags := TransportFlag(flags.GetEncoding())
	tflags.SetCompression(flags.GetCompression())
	cr := &clientRequest{flags: tflags, opaque: c.currOpaque, respch: respch}
	c.setRequest(c.currOpaque, cr)
	return cr, c.currOpaque
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
	txmsg := "%v trasmitting {%v,%x,%x,%T}\n"

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
		mtype, flags := msg[0].(uint16), msg[1].(TransportFlag)
		opaque := msg[2].(uint32)
		payload, respch := msg[3], msg[4].(chan []interface{})
		f_r, f_s := flags.IsRequest(), flags.IsStream()
		f_e := flags.IsEndStream()
		cr, known := c.getRequest(opaque)
		if f_r { // new request
			if f_s == false && f_e == false && known == false {
				log.Tracef(txmsg, prefix, "POST", flags, opaque, payload)

			} else if f_s && f_e && known == false {
				cr, opaque = c.addRequest(flags, respch)
				cr.rqst = true
				log.Tracef(txmsg, prefix, "RQST", flags, opaque, payload)

			} else if known {
				err = ErrorDuplicateRequest
				log.Errorf(txmsg, prefix, "DUPE", flags, opaque, payload)
				respch <- []interface{}{nil, err}

			} else {
				err = ErrorBadRequest
				log.Errorf(txmsg, prefix, "BADR", flags, opaque, payload)
				respch <- []interface{}{nil, err}
			}

		} else { // streaming request
		}
		if err == nil {
			conn.SetWriteDeadline(time.Now().Add(timeoutMs))
			if err := tpkt.Send(mtype, flags, opaque, payload); err != nil {
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
	rxmsg := "%v receiving {%x,%x,%x}\n"

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
		// will exit only when connection drop or server is closed.
		mtype, flags, opaque, payload, err := rpkt.Receive()
		if err == io.EOF {
			log.Infof("%v connection dropped: %v\n", prefix, err)
			break loop

		} else if err != nil {
			log.Errorf("%v connection failed: %v\n", prefix, err)
			break loop
		}
		cr, ok := c.getRequest(opaque)
		if ok {
			log.Tracef(rxmsg, prefix, flags, opaque, mtype)
			if cr.rqst {
				c.delRequest(opaque)
			}
			cr.respch <- []interface{}{payload, nil}
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
