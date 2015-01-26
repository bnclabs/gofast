//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "fmt"
import "net"
import "sync"
import "time"
import "io"
import "runtime/debug"

// PostHandler is callback from server to application
// to handle post-requests.
type PostHandler func(request interface{})

// RequestHandler is callback from server to application
// to handle single request and expects a single response,
// which application can post by calling `send` function.
type RequestHandler func(request interface{}, send ResponseSender)

// ResponseSender is callback-handler from server to application
// passed via RequestHandler. Application can use this
// function to post response message on the wire.
type ResponseSender func(mtype uint16, response interface{}, finish bool)

// StreamHandler is callback from server to application
// to start a bi-directional stream of messages between
// client and server. All subsequent message from client
// for same request (aka opaque) will be posted on `streamch`
// returned by the callback.
//
// [3]interface{}{mtype, payload, finish} -> streamch where,
//      mytype, describes the payload type.
//      payload, a single stream message.
//      finish, indicates that the other side whats to close.
//
// Application can stream one or more messages back to
// client by calling `send` function. Application should always
// signal end-of-response with {mtype, response, true}
type StreamHandler func(
	request interface{}, send ResponseSender) (streamch chan [3]interface{})

// Server handles incoming connections.
type Server struct {
	laddr string // address to listen
	// Handlers
	muHandler      sync.Mutex
	postHandler    PostHandler
	reqHandler     RequestHandler
	streamHandler  StreamHandler
	postHandlers   map[uint16]PostHandler    // map of msg-type -> handler
	reqHandlers    map[uint16]RequestHandler // map of msg-type -> handler
	streamHandlers map[uint16]StreamHandler  // map of msg-type -> handler
	// transport
	muTransport sync.Mutex
	encoders    map[TransportFlag]Encoder
	zippers     map[TransportFlag]Compressor
	// local fields
	muConn sync.Mutex
	lis    net.Listener
	conns  map[string]net.Conn // raddr -> active-connection
	killch chan bool
	// config params
	maxPayload     int           // for transport
	writeDeadline  time.Duration // timeout, in ms, while tx data to remote
	reqChanSize    int           // size of channel that mux requests
	streamChanSize int           // for each outstanding request
	log            Logger
	logPrefix      string
}

// NewServer creates server for fast communication.
// Subsequently application can register post, request
// or stream handlers with the server.
func NewServer(
	laddr string,
	config map[string]interface{},
	log Logger) (s *Server, err error) {

	if log == nil {
		log = SystemLog("server-logger")
	}
	s = &Server{
		laddr: laddr,
		// handlers
		postHandlers:   make(map[uint16]PostHandler),
		reqHandlers:    make(map[uint16]RequestHandler),
		streamHandlers: make(map[uint16]StreamHandler),
		// transport
		encoders: make(map[TransportFlag]Encoder),
		zippers:  make(map[TransportFlag]Compressor),
		// local fields
		killch: make(chan bool),
		conns:  make(map[string]net.Conn), // remoteaddr -> net.Conn
		// config
		maxPayload:     config["maxPayload"].(int),
		writeDeadline:  time.Duration(config["writeDeadline"].(int)),
		reqChanSize:    config["reqChanSize"].(int),
		streamChanSize: config["streamChanSize"].(int),
		log:            log,
		logPrefix:      fmt.Sprintf("GFST[%q]", laddr),
	}
	if s.lis, err = net.Listen("tcp", laddr); err != nil {
		log.Errorf("%v failed starting %v !!\n", s.logPrefix, err)
		return nil, err
	}

	go s.listener()
	log.Infof("%v started ...\n", s.logPrefix)
	return s, nil
}

// Close this instance of server.
func (s *Server) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("%v Close() crashed: %v\n", s.logPrefix, r)
			err = fmt.Errorf("%v", r)
			s.log.StackTrace(string(debug.Stack()))
		}
	}()

	s.muConn.Lock()
	defer s.muConn.Unlock()
	if s.lis != nil {
		s.lis.Close() // close listener daemon
		s.lis = nil
		// kill all connection handlers.
		close(s.killch)
		s.closeConnections() // acquires the lock and cleans up connection.
		s.conns = nil
		s.log.Infof("%v ... stopped\n", s.logPrefix)
	}
	return
}

// SetEncoder till set an encoder to encode and decode payload.
func (s *Server) SetEncoder(typ TransportFlag, encoder Encoder) {
	s.muTransport.Lock()
	defer s.muTransport.Unlock()
	s.encoders[typ] = encoder
}

// SetZipper will set a zipper type to compress and decompress payload.
func (s *Server) SetZipper(typ TransportFlag, zipper Compressor) {
	s.muTransport.Lock()
	defer s.muTransport.Unlock()
	s.zippers[typ] = zipper
}

// SetPostHandler default handler for post messages.
func (s *Server) SetPostHandler(handler PostHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.postHandler = handler
}

// SetPostHandlerFor specified message type.
func (s *Server) SetPostHandlerFor(mtype uint16, handler PostHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.postHandlers[mtype] = handler
}

// SetRequestHandler default handler for request messages.
func (s *Server) SetRequestHandler(handler RequestHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.reqHandler = handler
}

// SetRequestHandlerFor specified message type.
func (s *Server) SetRequestHandlerFor(mtype uint16, handler RequestHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.reqHandlers[mtype] = handler
}

// SetStreamHandler default handler for bi-directional streams.
func (s *Server) SetStreamHandler(handler StreamHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.streamHandler = handler
}

// SetStreamHandlerFor specified message type.
func (s *Server) SetStreamHandlerFor(mtype uint16, handler StreamHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.streamHandlers[mtype] = handler
}

// getPostHandler return post handler registered for `mtype`.
// - if handler is not registered for mtype return default
//   handler.
// - if default handler is not registered return nil.
func (s *Server) getPostHandler(mtype uint16) PostHandler {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	if handler, ok := s.postHandlers[mtype]; ok {
		return handler
	}
	return s.postHandler
}

// getRequestHandler return request handler registered for mtype.
// - if handler is not registered for mtype return default
//   handler.
// - if default handler is not registered return nil.
func (s *Server) getRequestHandler(mtype uint16) RequestHandler {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	if handler, ok := s.reqHandlers[mtype]; ok {
		return handler
	}
	return s.reqHandler
}

// getStreamHandler return stream handler registered for mtype.
// - if handler is not registered for mtype return default
//   handler.
// - if default handler is not registered return nil.
func (s *Server) getStreamHandler(mtype uint16) StreamHandler {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	if handler, ok := s.streamHandlers[mtype]; ok {
		return handler
	}
	return s.streamHandler
}

//-----------------
// request handling
//-----------------

// outstanding requests on a single connection.
type connRequests struct {
	mu       sync.Mutex
	requests map[uint32]*serverRequest // opaque -> *serverRequest
}

func (reqs *connRequests) addRequest(
	flags TransportFlag, opaque uint32) *serverRequest {

	// Note: Same encoding and compression as that of the initiating
	// request will be used while transporting a response for
	// that request.
	tflags := TransportFlag(flags.GetEncoding())
	tflags.SetCompression(flags.GetCompression())
	sr := &serverRequest{flags: tflags, opaque: opaque}
	reqs.setRequest(opaque, sr)
	return sr
}

func (reqs *connRequests) setRequest(opaque uint32, sr *serverRequest) {
	reqs.mu.Lock()
	defer reqs.mu.Unlock()
	reqs.requests[opaque] = sr
}

func (reqs *connRequests) getRequest(opaque uint32) (*serverRequest, bool) {
	reqs.mu.Lock()
	defer reqs.mu.Unlock()
	sr, ok := reqs.requests[opaque]
	return sr, ok
}

func (reqs *connRequests) delRequest(opaque uint32) {
	reqs.mu.Lock()
	defer reqs.mu.Unlock()
	delete(reqs.requests, opaque)
}

type serverRequest struct {
	flags    TransportFlag
	opaque   uint32
	rqst     bool
	strm     bool
	streamch chan [3]interface{}
}

//--------------------
// Connection handling
//--------------------

// add a new connection to server
func (s *Server) addConn(conn net.Conn) {
	s.muConn.Lock()
	defer s.muConn.Unlock()
	if s.conns != nil { // server has already closed !
		s.conns[conn.RemoteAddr().String()] = conn
	}
}

// close all active connections, caller should protect it with mutex.
func (s *Server) closeConnections() {
	closeMsg := "%v close connection with client %v\n"
	for raddr, conn := range s.conns {
		s.log.Infof(closeMsg, s.logPrefix, raddr)
		conn.Close()
	}
}

func (s *Server) isClosed() bool {
	select {
	case <-s.killch:
		return true
	default:
	}
	return false
}

// receive requests from remote, when this function returns
// the connection is expected to be closed.
func (s *Server) doReceive(conn net.Conn, reqch chan<- []interface{}) {
	raddr, log, prefix := conn.RemoteAddr(), s.log, s.logPrefix

	rpkt := s.newTransport(conn)
	log.Debugf("%v doReceive() from %q ...\n", prefix, raddr)
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("%v doReceive() %q crashed: %v\n", prefix, raddr, r)
			StackTrace(string(debug.Stack()))
		}
		close(reqch)
	}()

loop:
	for {
		// Note: receive on connection shall work without timeout.
		// will exit only when connection drops or server is closed.
		mtype, flags, opaque, payload, err := rpkt.Receive()
		if err == io.EOF {
			log.Infof("%v connection %q dropped: %v\n", prefix, raddr, err)
			break loop

		} else if s.isClosed() {
			log.Infof("%v server stopped: %v\n", prefix, err)
			break loop

		} else if err != nil {
			log.Errorf("%v connection %q failed: %v\n", prefix, raddr, err)
			break loop
		}
		select {
		case reqch <- []interface{}{mtype, flags, opaque, payload}:
		case <-s.killch:
			break loop
		}
	}
}

// handle a single connection from remote.
func (s *Server) handleConnection(conn net.Conn, reqch chan []interface{}) {
	log, prefix, raddr := s.log, s.logPrefix, conn.RemoteAddr()
	tpkt := s.newTransport(conn)
	log.Debugf("%v handleConnection() ...\n", prefix)
	rxmsg := "%v {%s,%x,%x,%x} <- %q\n"
	txmsg := "%v {%x,%x,%x,%T} -> %q\n"

	defer func() {
		if r := recover(); r != nil {
			msg := "%v handleConnection() %q crashed: %v\n"
			log.Errorf(msg, prefix, raddr, r)
			StackTrace(string(debug.Stack()))
		}
		conn.Close()
		log.Debugf("%v handleConnection(%v) exiting ...\n", prefix, raddr)
	}()

	// transport buffer for transmission
	respch := make(chan []interface{}, s.streamChanSize)
	timeoutMs := s.writeDeadline * time.Millisecond
	srs := connRequests{requests: make(map[uint32]*serverRequest)}

loop:
	for {
		select {
		case req, ok := <-reqch:
			if !ok { // doReceive() has closed
				break loop
			}
			mtype, flags := req[0].(uint16), req[1].(TransportFlag)
			opaque, payload := req[2].(uint32), req[3]
			// flags
			f_r, f_s := flags.IsRequest(), flags.IsStream()
			f_e := flags.IsEndStream()
			sr, known := srs.getRequest(opaque)
			if f_r { // new request
				if f_s == false && f_e == false && known == false {
					log.Tracef(rxmsg, prefix, "POST", flags, opaque, mtype, raddr)
					if callb := s.getPostHandler(mtype); callb != nil {
						callb(payload)
					}

				} else if f_s && f_e && known == false {
					log.Tracef(rxmsg, prefix, "RQST", flags, opaque, mtype, raddr)
					if callb := s.getRequestHandler(mtype); callb != nil {
						sr = srs.addRequest(flags, opaque)
						sr.rqst = true
						callb(payload, makeRespSender(opaque, respch))
					}

				} else if f_s && f_e == false && known == false {
					log.Tracef(rxmsg, prefix, "STRM", flags, opaque, mtype, raddr)
					if callb := s.getStreamHandler(mtype); callb != nil {
						sr = srs.addRequest(flags, opaque)
						sr.strm = true
						sr.streamch = callb(payload, makeRespSender(opaque, respch))
					}
				}

			} else if known && sr.strm { // stream-request that we already know
				if sr.streamch != nil {
					if f_e { // this is the last message.
						log.Tracef(rxmsg, prefix, "ENDS", flags, opaque, mtype, raddr)
						sr.streamch <- [3]interface{}{mtype, payload, true}
						sr.streamch = nil

					} else {
						log.Tracef(rxmsg, prefix, "STRM", flags, opaque, mtype, raddr)
						sr.streamch <- [3]interface{}{mtype, payload, false}
					}

				} else { // incoming stream has already been closed.
					log.Errorf(rxmsg, prefix, "STRM", flags, opaque, mtype, raddr)
				}

			} else {
				log.Errorf(rxmsg, prefix, "UNKN", flags, opaque, mtype, raddr)
			}

		case resp := <-respch:
			mtype, opaque := resp[0].(uint16), resp[1].(uint32)
			response, finish := resp[2], resp[3].(bool)
			sr, known := srs.getRequest(opaque)
			if known { // response is sent back for known requests
				flags := sr.flags
				if sr.rqst || finish {
					flags = flags.SetEndStream()
					srs.delRequest(opaque)
				} else if sr.strm {
					flags = flags.SetStream()
				}
				log.Tracef(txmsg, prefix, flags, opaque, mtype, response, raddr)
				conn.SetWriteDeadline(time.Now().Add(timeoutMs))
				if err := tpkt.Send(mtype, flags, opaque, response); err != nil {
					break loop
				}

			} else {
				log.Errorf(txmsg, prefix, 0, opaque, mtype, response, raddr)
			}

		case <-s.killch:
			break loop
		}
	}
}

// go-routine to listen for new connections,
// if this routine exits server is shutdown.
func (s *Server) listener() {
	log := s.log
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("%v listener() crashed: %v\n", s.logPrefix, r)
			StackTrace(string(debug.Stack()))
		}
		go s.Close()
	}()

	for {
		if conn, err := s.lis.Accept(); err == nil {
			s.addConn(conn)

			// start a receive routine.
			reqch := make(chan []interface{}, s.reqChanSize)
			go s.doReceive(conn, reqch)

			// handle both socket transmission and callback dispatching
			go s.handleConnection(conn, reqch)

		} else {
			if e, ok := err.(*net.OpError); ok && e.Op != "accept" {
				panic(err)
			}
			break
		}
	}
}

//----------------
// local functions
//----------------

func (s *Server) newTransport(conn net.Conn) *TransportPacket {
	s.muTransport.Lock()
	defer s.muTransport.Unlock()
	pkt := NewTransportPacket(conn, s.maxPayload, s.log)
	for flag, encoder := range s.encoders {
		pkt.SetEncoder(flag.GetEncoding(), encoder)
	}
	for flag, zipper := range s.zippers {
		pkt.SetZipper(flag.GetCompression(), zipper)
	}
	return pkt
}

func makeRespSender(opaque uint32, respch chan<- []interface{}) ResponseSender {
	return func(mtype uint16, response interface{}, finish bool) {
		respch <- []interface{}{mtype, opaque, response, finish}
	}
}
