//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "net"
import "sync"
import "time"
import "fmt"
import "io"
import "runtime/debug"

// ResponseReceiver is callback-handler from client to application.
type ResponseReceiver func(
	mtype uint16, opaque uint32, response interface{}, finish bool)

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
// Asynchronous call. `flags` can be set with a supported
// encoding and compression type.
func (c *Client) Post(
	flags TransportFlag, mtype uint16, payload interface{}) error {

	respch := make(chan []interface{}, 1)
	flags = flags.ClearStream().ClearEndStream().SetRequest()
	cmd := []interface{}{
		mtype, flags, NoOpaque, payload, ResponseReceiver(nil), respch}
	resp, err := failsafeOp(c.muxch, respch, cmd, c.finch)
	if err := opError(err, resp, 1); err != nil {
		return err
	}
	return nil
}

// Request server and wait for a single response from server
// and return response from server. `request` is actual request,
// `mtype` is 16-bit value that can be used to interpret payload,
// for `flags` refer to transport spec.
func (c *Client) Request(
	flags TransportFlag,
	mtype uint16,
	request interface{}) (response interface{}, err error) {

	respch := make(chan []interface{}, 1)
	callbch := make(chan interface{}, 1)
	callb := func(_ uint16, _ uint32, response interface{}, _ bool) {
		callbch <- response
	}
	flags = flags.SetRequest().SetStream().SetEndStream()
	cmd := []interface{}{
		mtype, flags, NoOpaque, request, ResponseReceiver(callb), respch}
	resp, err := failsafeOp(c.muxch, respch, cmd, c.finch)
	if err := opError(err, resp, 1); err != nil {
		return nil, err
	}
	response = <-callbch
	return response, nil
}

// RequestStream server for bi-directional streaming with
// server.  `mtype` is 16-bit value that can be used to interpret
// payload, `request` is actual request, `callb` is callback from
// client to application to handle incoming response messages,
// for `flags` refer to transport spec. same encoding and
// compression will be used on all subsequent messages.
//
// when no error, return the opaque-id that can be used
// used for subsequent calls on the stream.
func (c *Client) RequestStream(
	flags TransportFlag,
	mtype uint16,
	request interface{},
	callb ResponseReceiver) (opaque uint32, err error) {

	respch := make(chan []interface{}, 1)
	flags = flags.SetRequest().SetStream()
	cmd := []interface{}{mtype, flags, NoOpaque, request, callb, respch}
	resp, err := failsafeOp(c.muxch, respch, cmd, c.finch)
	if err := opError(err, resp, 1); err != nil {
		return NoOpaque, err
	}
	return resp[0].(uint32), nil
}

// Stream a message associated to an original request
// identified by `opaque`. `finish` will end the streaming
// from the client side.
func (c *Client) Stream(
	opaque uint32, mtype uint16, msg interface{}, finish bool) error {

	respch := make(chan []interface{}, 1)
	flags := TransportFlag(0)
	if finish {
		flags = flags.SetEndStream()
	}
	cmd := []interface{}{
		mtype, flags, opaque, msg, ResponseReceiver(nil), respch}
	resp, err := failsafeOp(c.muxch, respch, cmd, c.finch)
	if err := opError(err, resp, 1); err != nil {
		return err
	}
	return nil
}

// StreamClose will close a stream on client side. Alternately stream
// can also be closed via Stream() call by setting the `finish` flag.
func (c *Client) StreamClose(opaque uint32) error {
	respch := make(chan []interface{}, 1)
	flags := TransportFlag(0).SetEndStream()
	cmd := []interface{}{
		MtypeEmpty, flags, opaque, nil, ResponseReceiver(nil), respch}
	resp, err := failsafeOp(c.muxch, respch, cmd, c.finch)
	if err := opError(err, resp, 1); err != nil {
		return err
	}
	return nil
}

//-----------------
// request handling
//-----------------

type clientRequest struct {
	flags  TransportFlag
	opaque uint32
	rqst   bool
	strm   bool
	callb  ResponseReceiver
}

func (c *Client) addRequest(
	flags TransportFlag, callb ResponseReceiver) (*clientRequest, uint32) {

	// Note: Request streams cannot switch encoding and
	// compression after the request is initiated.
	c.currOpaque++
	if c.currOpaque > 0x7FFFFFFF { // TODO: no magic numbers
		c.currOpaque = 1
	}
	tflags := TransportFlag(flags.GetEncoding())
	tflags.SetCompression(flags.GetCompression())
	cr := &clientRequest{flags: tflags, opaque: c.currOpaque, callb: callb}
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
	txmsg := "%v transmitting {%v,%x,%x,%T}\n"

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
		mtype, flags, opaque, payload, callb, respch := d(msg)
		cr, known := c.getRequest(opaque)
		if known && cr.strm { // streaming message
			if flags.IsEndStream() {
				flags = cr.flags.SetEndStream()
				log.Tracef(txmsg, prefix, "ENDS", flags, opaque, payload)
			} else {
				flags = cr.flags.SetStream()
				log.Tracef(txmsg, prefix, "STRM", flags, opaque, payload)
			}
			respch <- []interface{}{nil, nil}

		} else if f_r := flags.IsRequest(); f_r { // new request
			f_s, f_e := flags.IsStream(), flags.IsEndStream()
			if f_s == false && f_e == false && known == false {
				log.Tracef(txmsg, prefix, "POST", flags, opaque, payload)
				respch <- []interface{}{nil, nil}

			} else if f_s && f_e && known == false {
				log.Tracef(txmsg, prefix, "RQST", flags, opaque, payload)
				cr, opaque = c.addRequest(flags, callb)
				cr.rqst = true
				respch <- []interface{}{nil, nil}

			} else if f_s && f_e == false && known == false {
				cr, opaque = c.addRequest(flags, callb)
				log.Tracef(txmsg, prefix, "STRM", flags, opaque, payload)
				cr.strm = true
				respch <- []interface{}{opaque, nil}

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
			err = ErrorUnknownStream
			log.Errorf(txmsg, prefix, "UNKN", flags, opaque, payload)
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
	rxmsg := "%v receiving {%v,%x,%x,%x}\n"

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
			finish := flags.IsEndStream()
			if cr.rqst {
				log.Tracef(rxmsg, prefix, "RQST", flags, opaque, mtype)
				cr.callb(mtype, opaque, payload, finish)
			} else if cr.strm {
				log.Tracef(rxmsg, prefix, "STRM", flags, opaque, mtype)
				cr.callb(mtype, opaque, payload, finish)
			} else {
				log.Errorf(rxmsg, prefix, "BADT", flags, opaque, mtype)
			}
			if finish {
				c.delRequest(opaque)
			}

		} else { // ignore reponses if opaque not found.
			log.Errorf(rxmsg, prefix, "BADO", flags, opaque, mtype)
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

func d(msg []interface{}) (
	uint16, TransportFlag, uint32, interface{}, ResponseReceiver,
	chan []interface{}) {
	var respch chan []interface{}
	mtype, flags := msg[0].(uint16), msg[1].(TransportFlag)
	opaque := msg[2].(uint32)
	payload, callb := msg[3], msg[4].(ResponseReceiver)
	respch = msg[5].(chan []interface{})
	return mtype, flags, opaque, payload, callb, respch
}
