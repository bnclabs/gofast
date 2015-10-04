//  Copyright (c) 2014 Couchbase, Inc.

// Package gofast implements a high performance symmetric protocol
// for on the wire data transport.
//
// message-format:
//
//	  A message is encoded as finite length CBOR map with predefined list
//    of keys, for example, "version", "compression", "type",
//    "data" etc... keys are typically encoded as uint16 numbers so that
//    they can be efficiently packed. This implies that each of the
//    predefined keys shall be assigned a unique number.
//
// packet-format:
//
//	  a single block of binary blob in CBOR format, transmitted
//    from client to server or server to client:
//
//		tag1 |         payload1			 |
//			 | tag2 |      payload2		 |
//					| tag3 |   payload3  |
//						   | tag 4 | msg |
//
//
//    * payload shall always be encoded as CBOR byte array.
//	  * tags are uint64 numbers that will either be prefixed
//      to payload or msg.
//
//    In the above example:
//
//	  * tag1, will always be a opaque number falling within a
//      reserved tag-space called opaque-space.
//    * tag2, tag3 can be one of the values predefined by gofast.
//    * the final embedded tag, in this case tag4, shall always
//      be tagMsg (value 37).
//
// every exchange shall be initiated by the client, exchange can be:
//
//	   post-request, client post a packet and expects no response:
//
//			| 0xd9f7 | packet |
//
//	   request-response, client make a request and expects a
//     single response:
//
//			| 0xd9f7 | 0x9f | packet | 0xff |
//
//	   bi-directional streaming, where client and server will have to close
//     the stream by sending a 0xff.
//
//			| 0xd9f7 | 0x9f | packet1 |
//							| 0xd9f7 | packet2 |
//							...
//							| 0xd9f7 | packetN | 0xff |
//
// except of post-request, the exchange between client and server is always
// symmetrical.
package gofast

import "encoding/binary"
import "net"
import "io"
import "fmt"

// Transporter interface to send and receive packets.
// APIs are not thread safe.
type Transporter interface { // facilitates unit testing
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

// Encoder interface to Encode()/Decode() payload object to
// raw bytes.
//
// Encode callback for Send() packet. Encoder can use `out`
// buffer to convert the payload, either case it shall return
// a valid output slice. Return buffer as byte-slice, may be
// a reference into `out` array, with exact length.
//
// Decode callback while Receive() packet.
type Encoder interface {
	Encode(
		flags TransportFlag, opaque uint32, payload interface{},
		out []byte) (data []byte, err error)

	Decode(
		mtype uint16, flags TransportFlag, opaque uint32,
		data []byte) (payload interface{}, err error)
}

// Compressor interface inflate and deflate raw bytes before
// sending on wire.
//
// Zip callback for Send() packet. Zip can use `out`
// buffer to send back compressed data, in either case it shall
// return a valid output slice. Return buffer as byte-slice,
// may be a reference into `out` array, with exact length.
//
// Unzip callback while Receive() packet. Unzip can use `out`
// buffer to send back compressed data. Returns buffer as
// byte-slice, may be a reference into `out` array, with exact
// length.
type Compressor interface {
	Zip(in, out []byte) (data []byte, err error)
	Unzip(in, out []byte) (data []byte, err error)
}

// TransportPacket to send and receive mutation packets between
// router and downstream client. Not thread safe.
type TransportPacket struct {
	conn      Transporter
	flags     TransportFlag
	bufEnc    []byte
	bufComp   []byte
	encoders  map[TransportFlag]Encoder
	zippers   map[TransportFlag]Compressor
	log       Logger
	isHealthy bool
	logPrefix string
}

// NewTransportPacket creates a new transporter on a single connection
// to frame, encode and compress payload before sending it to remote and
// deframe, decompress, decode while receiving payload from remote.
//   - if `log` argument is nil, builtin logger will be used.
//   - `buflen` defines the maximum size of the packet, decompressed
//     payload shall not exceed this size.
func NewTransportPacket(
	conn Transporter, buflen int, log Logger) *TransportPacket {

	if log == nil {
		log = SystemLog("transport-logger")
	}

	laddr, raddr := conn.LocalAddr(), conn.RemoteAddr()
	pkt := &TransportPacket{
		conn:      conn,
		bufEnc:    make([]byte, buflen),
		bufComp:   make([]byte, buflen),
		encoders:  make(map[TransportFlag]Encoder),
		zippers:   make(map[TransportFlag]Compressor),
		log:       log,
		isHealthy: true,
		logPrefix: fmt.Sprintf("CONN[%v<->%v]", laddr, raddr),
	}
	return pkt
}

// Send payload to the other end. Payload is defined by,
// {message-type (mtype), flags, opaque}
//   - flags specify encoding, compression and streaming semantics.
//   - opaque can be used for concurrent requests on same connection.
//
// caller can check return error for io.EOF to detect connection
// drops.
func (pkt *TransportPacket) Send(
	mtype uint16, flags TransportFlag, opaque uint32,
	payload interface{}) error {

	prefix, log := pkt.logPrefix, pkt.log

	// encode
	encoder, ok := pkt.encoders[flags.GetEncoding()]
	if !ok {
		log.Errorf("%v (flags %x) Send() unknown encoder\n", prefix, flags)
		return ErrorEncoderUnknown
	}
	buf, err := encoder.Encode(flags, opaque, payload, pkt.bufEnc)
	if err != nil {
		log.Errorf("%v (flags %x) Send() encode: %v\n", prefix, flags, err)
		return err
	}

	// compress
	if flags.GetCompression() != CompressionNone {
		compressor, ok := pkt.zippers[flags.GetCompression()]
		if !ok {
			log.Errorf("%v (flags %x) Send() unknown zipper\n", prefix, flags)
			return ErrorZipperUnknown
		}
		buf, err = compressor.Zip(buf, pkt.bufComp[pktDataOffset:])
		if err != nil {
			log.Errorf("%v (flags %x) Send() zipper: %v\n", prefix, flags, err)
			return err
		}
	}

	// first send header
	frameHdr(mtype, uint16(flags), opaque, uint32(len(buf)),
		pkt.bufComp[:pktDataOffset])
	if n, err := pkt.conn.Write(pkt.bufComp[:pktDataOffset]); err != nil {
		if err == io.EOF {
			log.Infof("%v: connection dropped %v\n", prefix, err)
			return err
		}
		log.Errorf("%v (flags %x) Send() failed: %v\n", prefix, flags, err)
		return ErrorPacketWrite

	} else if l := int(pktDataOffset); n != l {
		msg := "%v (flags %x) Send() wrote %v(%v) bytes\n"
		log.Errorf(msg, prefix, flags, l, n)
		return ErrorPacketWrite
	}

	// then send payload
	if n, err := pkt.conn.Write(buf); err != nil {
		if err == io.EOF {
			log.Infof("%v: connection dropped %v\n", prefix, err)
			return err
		}
		log.Errorf("%v (flags %x) Send() failed: %v\n", prefix, flags, err)
		return ErrorPacketWrite

	} else if l := len(buf); n != l {
		msg := "%v (flags %x) Send() wrote %v(%v) bytes\n"
		log.Errorf(msg, prefix, flags, l, n)
		return ErrorPacketWrite
	}
	log.Tracef("%v {%x,%x,%x} -> wrote %v bytes\n",
		prefix, mtype, flags, opaque, len(buf)+int(hdrLen))
	return nil
}

// Receive payload from remote. Payload is defined by,
// {message-type (mtype), flags, opaque}
//   - flags specify encoding, compression and streaming semantics.
//   - opaque can be used for concurrent requests on same connection.
//
// caller can check return error for io.EOF to detect connection
// drops.
func (pkt *TransportPacket) Receive() (
	mtype uint16, flags TransportFlag, opaque uint32,
	payload interface{}, err error) {

	prefix, log := pkt.logPrefix, pkt.log

	// read and de-frame header
	if err = fullRead(pkt.conn, pkt.bufComp[:pktDataOffset]); err != nil {
		if err == io.EOF {
			log.Infof("%v: connection dropped %v\n", prefix, err)
			return
		}
		log.Errorf("%v Receive() packet failed: %v\n", prefix, err)
		err = ErrorPacketRead
		return
	}
	mtype, f, opaque, ln := deframeHdr(pkt.bufComp[:pktDataOffset])
	if l, maxLen := (uint32(hdrLen) + ln), cap(pkt.bufComp); l > uint32(maxLen) {
		log.Errorf("%v Receive() packet %v > %v\n", prefix, l, maxLen)
		err = ErrorPacketOverflow
		return
	}

	// read payload
	buf := pkt.bufComp[pktDataOffset : uint32(pktDataOffset)+ln]
	if err = fullRead(pkt.conn, buf); err != nil {
		if err == io.EOF {
			log.Infof("%v: connection dropped %v\n", prefix, err)
			return
		}
		log.Errorf("%v Receive() packet failed: %v\n", prefix, err)
		err = ErrorPacketRead
		return
	}
	log.Tracef("%v {%x,%x,%x,%x} <- read %v bytes\n",
		prefix, mtype, f, opaque, ln, ln+uint32(hdrLen))

	flags = TransportFlag(f)

	// de-compress
	if flags.GetCompression() != CompressionNone {
		compressor, ok := pkt.zippers[flags.GetCompression()]
		if !ok {
			log.Errorf("%v (flags %x) Receive() unknown zipper\n", prefix, flags)
			err = ErrorZipperUnknown
			return
		}
		buf, err = compressor.Unzip(buf, pkt.bufEnc)
		if err != nil {
			log.Errorf("%v (flags %x) Receive() zipper: %v\n", prefix, flags, err)
			return
		}
	}

	// decode
	encoder, ok := pkt.encoders[flags.GetEncoding()]
	if !ok {
		log.Errorf("%v (flags %x) Receive() unknown encoder\n", prefix, flags)
		err = ErrorEncoderUnknown
		return
	}
	payload, err = encoder.Decode(mtype, flags, opaque, buf)
	if err != nil {
		log.Errorf("%v (flags %x) Receive() encoder: %v\n", prefix, flags, err)
	}
	return
}

//------------------
// Transport framing
//------------------

// packet field offset and size in bytes
const (
	pktTypeOffset byte = 0
	pktTypeSize   byte = 2 // bytes
	pktFlagOffset byte = pktTypeOffset + pktTypeSize
	pktFlagSize   byte = 2 // bytes
	pktOpqOffset  byte = pktFlagOffset + pktFlagSize
	pktOpqSize    byte = 4 // bytes
	pktLenOffset  byte = pktOpqOffset + pktOpqSize
	pktLenSize    byte = 4
	pktDataOffset byte = pktLenOffset + pktLenSize

	hdrLen byte = pktTypeSize + pktFlagSize + pktOpqSize + pktLenSize
)

func frameHdr(mtype, flags uint16, opaque uint32, datalen uint32, hdr []byte) {
	binary.BigEndian.PutUint16(hdr[pktTypeOffset:pktFlagOffset], mtype)
	binary.BigEndian.PutUint16(hdr[pktFlagOffset:pktOpqOffset], flags)
	binary.BigEndian.PutUint32(hdr[pktOpqOffset:pktLenOffset], opaque)
	binary.BigEndian.PutUint32(hdr[pktLenOffset:pktDataOffset], datalen)
}

func deframeHdr(hdr []byte) (mtype, flags uint16, opaque, datalen uint32) {
	mtype = binary.BigEndian.Uint16(hdr[pktTypeOffset:pktFlagOffset])
	flags = binary.BigEndian.Uint16(hdr[pktFlagOffset:pktOpqOffset])
	opaque = binary.BigEndian.Uint32(hdr[pktOpqOffset:pktLenOffset])
	datalen = binary.BigEndian.Uint32(hdr[pktLenOffset:pktDataOffset])
	return
}

//----------------
// local functions
//----------------

// read len(buf) bytes from `conn`.
func fullRead(conn Transporter, buf []byte) error {
	size, start := 0, 0
	for size < len(buf) {
		n, err := conn.Read(buf[start:])
		if err != nil {
			return err
		}
		size += n
		start += n
	}
	return nil
}
