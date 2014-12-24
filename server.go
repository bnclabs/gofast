package gofast

import "fmt"
import "net"
import "sync"
import "time"
import "io"
import "runtime/debug"

// PostHandler is callback from transport to application to handle
// post-requests.
type PostHandler func(opaque uint32, request interface{})

// from client and post response message(s) on `respch`
// channel, until `quitch` is closed.
// - if `respch` is empty, then client is not expecting a response.
// - if `quitch` is empty, then callb() should forget the request after
//   sending back a single response.
// - if no more response to be sent, then following message should be posted
//      []interface{}{opaque, nil, true}
//   after posting msg, call-back shall return false.
//   after posting this message, call-back will not be called for same opaque
//   unless otherwise it is for a new request.
// - if payload is last response, then following message should be posted
//      []interface{}{opaque, payload, true}
// - if client does not want to receive anymore response for that request,
//   `quitch` shall be closed.

// RequestHandler is callback from transport to application to handle
// single request and expects a single response, which shall be posted
// on `respch`.
// response format from callback -- []interface{}{opaque, payload}
// where,
//      opaque, same as the one received from request.
type RequestHandler func(
	opaque uint32, req interface{}, respch chan []interface{})

// StreamHandler is callback from transport to application to handle
// bi-directional stream of messages between client and server. A
// stream of request messages, with same opaque, shall be posted
// on `streamch` returned by the callback.
// request format on streamch -- []interface{}{opaque, payload, finish}
// response format from callback -- []interface{}{opaque, payload, finish}
// where,
//      opaque, same as the one received from initiating request.
//      payload, a single stream message
//      finish, indicates that the other side whats to close.
type StreamHandler func(
	opaque uint32, req interface{},
	respch chan []interface{}) (streamch chan []interface{})

// Server handles incoming connections.
type Server struct {
	laddr string // address to listen
	// Handlers
	muHandler      sync.Mutex
	postHandler    PostHandler
	reqHandler     RequestHandler
	streamHandler  StreamHandler
	postHandlers   map[uint32]PostHandler    // map of opaque -> handler
	reqHandlers    map[uint32]RequestHandler // map of opaque -> handler
	streamHandlers map[uint32]StreamHandler  // map of opaque -> handler
	// transport
	muTransport sync.Mutex
	encoders    map[TransportFlag]Encoder
	zippers     map[TransportFlag]Compressor
	requests    map[uint32]*serverRequest // map of opaque -> serverRequest
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

// NewServer creates a new server for fast communication.
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
		postHandlers:   make(map[uint32]PostHandler),
		reqHandlers:    make(map[uint32]RequestHandler),
		streamHandlers: make(map[uint32]StreamHandler),
		// transport
		encoders: make(map[TransportFlag]Encoder),
		zippers:  make(map[TransportFlag]Compressor),
		requests: make(map[uint32]*serverRequest), // opaque -> serverReq.
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

	s.closeConnections() // acquires the lock and cleans up connection.

	s.muConn.Lock()
	defer s.muConn.Unlock()
	if s.lis != nil {
		s.lis.Close() // close listener daemon
		s.lis = nil
		// kill all connection handlers.
		close(s.killch)
		s.log.Infof("%v ... stopped\n", s.logPrefix)
	}
	return
}

// SetPostHandler default handler for post messages.
func (s *Server) SetPostHandler(handler PostHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.postHandler = handler
}

// SetPostHandlerFor specified opaques.
func (s *Server) SetPostHandlerFor(opaque uint32, handler PostHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.postHandlers[opaque] = handler
}

// SetRequestHandler default handler for request messages.
func (s *Server) SetRequestHandler(handler RequestHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.reqHandler = handler
}

// SetRequestHandlerFor specified opaques.
func (s *Server) SetRequestHandlerFor(opaque uint32, handler RequestHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.reqHandlers[opaque] = handler
}

// SetStreamHandler default handler for bi-directional streams.
func (s *Server) SetStreamHandler(handler StreamHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.streamHandler = handler
}

// SetStreamHandlerFor specified opaques.
func (s *Server) SetStreamHandlerFor(opaque uint32, handler StreamHandler) {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	s.streamHandlers[opaque] = handler
}

// getPostHandler return handler for post request, registered
// for opaque. If no handler registered for opaque return default
// handler.
func (s *Server) getPostHandler(opaque uint32) PostHandler {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	if handler, ok := s.postHandlers[opaque]; ok {
		return handler
	}
	return s.postHandler
}

// getRequestHandler return handler for single response, registered
// for opaque. If no handler registered for opaque return default
// handler.
func (s *Server) getRequestHandler(opaque uint32) RequestHandler {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	if handler, ok := s.reqHandlers[opaque]; ok {
		return handler
	}
	return s.reqHandler
}

// getStreamHandler return handler for bi-directional streaming,
// registered for opaque. If no handler registered for opaque
// return default handler.
func (s *Server) getStreamHandler(opaque uint32) StreamHandler {
	s.muHandler.Lock()
	defer s.muHandler.Unlock()
	if handler, ok := s.streamHandlers[opaque]; ok {
		return handler
	}
	return s.streamHandler
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

//--------------------
// Connection handling
//--------------------

// go-routine to listen for new connections, if this routine goes
// down server is shutdown.
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
	s.muConn.Lock()
	defer s.muConn.Unlock()
	for raddr, conn := range s.conns {
		s.log.Infof("%v close connection with client %v\n", s.logPrefix, raddr)
		conn.Close()
	}
	s.conns = nil
}

//-----------------
// request handling
//-----------------

type serverRequest struct {
	flags    TransportFlag
	opaque   uint32
	rqst     bool
	strm     bool
	streamch chan []interface{}
}

func (s *Server) addRequest(flags TransportFlag, opaque uint32) *serverRequest {
	// Note: Same encoding and compression as that of the initiating
	// request will be used while transporting a response for
	// that request.
	tflags := TransportFlag(flags.GetEncoding())
	tflags.SetCompression(flags.GetCompression())
	sr := &serverRequest{flags: tflags, opaque: opaque}
	s.setRequest(opaque, sr)
	return sr
}

func (s *Server) setRequest(opaque uint32, cr *serverRequest) {
	s.muTransport.Lock()
	defer s.muTransport.Unlock()
	s.requests[opaque] = cr
}

func (s *Server) getRequest(opaque uint32) (*serverRequest, bool) {
	s.muTransport.Lock()
	defer s.muTransport.Unlock()
	cr, ok := s.requests[opaque]
	return cr, ok
}

func (s *Server) delRequest(opaque uint32) {
	s.muTransport.Lock()
	defer s.muTransport.Unlock()
	delete(s.requests, opaque)
}

//--------------------
// Connection handling
//--------------------

// handle connection request. connection might be kept open in client's
// connection pool.
func (s *Server) handleConnection(conn net.Conn, reqch chan []interface{}) {
	log, prefix, raddr := s.log, s.logPrefix, conn.RemoteAddr()
	tpkt := s.newTransport(conn)
	log.Debugf("%v handleConnection() ...\n", prefix)
	rxmsg := "%v on connection %q received %s {%x,%x,%x}\n"
	txmsg := "%v on connection %q transmitting response {%x,%x,%T}\n"

	defer func() {
		if r := recover(); r != nil {
			log.Errorf(
				"%v handleConnection() %q crashed: %v\n", prefix, raddr, r)
			StackTrace(string(debug.Stack()))
		}
		conn.Close()
		log.Debugf("%v handleConnection(%v) exiting ...\n", prefix, raddr)
	}()

	// transport buffer for transmission
	respch := make(chan []interface{}, s.streamChanSize)
	timeoutMs := s.writeDeadline * time.Millisecond

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
			_, known := s.getRequest(opaque)
			if f_r { // new request
				if f_s == false && f_e == false && known == false {
					// post and forget it.
					log.Tracef(rxmsg, prefix, raddr, "POST", flags, opaque, mtype)
					if callb := s.getPostHandler(opaque); callb != nil {
						callb(opaque, payload)
					}

				} else if f_s && f_e && known == false {
					// request and wait for cleanup.
					log.Tracef(rxmsg, prefix, raddr, "RQST", flags, opaque, mtype)
					if callb := s.getRequestHandler(opaque); callb != nil {
						callb(opaque, payload, respch)
						sr := s.addRequest(flags, opaque)
						sr.rqst = true
					}
				}

			} else { // ignore
			}

		case resp := <-respch:
			opaque, response := resp[0].(uint32), resp[1]
			sr, known := s.getRequest(opaque)
			flags := sr.flags
			if sr.rqst {
				flags = flags.SetEndStream()
				s.delRequest(opaque)
			}
			if known {
				log.Tracef(txmsg, prefix, raddr, flags, opaque, response)
				conn.SetWriteDeadline(time.Now().Add(timeoutMs))
				if err := tpkt.Send(flags, opaque, response); err != nil {
					break loop
				}
			}

		case <-s.killch:
			break loop
		}
	}
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
		// will exit only when remote fails or server is closed.
		mtype, flags, opaque, payload, err := rpkt.Receive()
		if err != nil {
			if err != io.EOF { // TODO check whether connection closed
				log.Errorf("%v client %q exited: %v\n", prefix, raddr, err)
			}
			break loop
		}
		select {
		case reqch <- []interface{}{mtype, flags, opaque, payload}:
		case <-s.killch:
			break loop
		}
	}
}
