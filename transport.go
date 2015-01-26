//  Copyright (c) 2014 Couchbase, Inc.

// Package gofast implements a high performance symmetric protocol
// for on the wire data transport.
//
//    0               8               16              24            31
//    +---------------+---------------+---------------+---------------+
//    |         Message type          |             Flags             |
//    +---------------+---------------+---------------+---------------+
//    |                     Opaque value (uint32)                     |
//    +---------------+---------------+---------------+---------------+
//    |                   payload-length (uint32)                     |
//    +---------------+---------------+---------------+---------------+
//    |                        payload ....                           |
//    +---------------+---------------+---------------+---------------+
//
// mtype-field:
//   * field states the type of `payload` carried by the packet.
//   * values shall always start from 1.
//   * values from 0xF000 onwards are reserved by protocol.
//   * value 0xFFFF used to advertise receiver's buffer size to transmitter.
//
// flags-field:
//
//           +---------------+---------------+
//       byte|       0       |       1       |
//           +---------------+---------------+
//       bits|0 1 2 3 4 5 6 7|0 1 2 3 4 5 6 7|
//           +-------+-------+---------------+
//          0| COMP. |  ENC. |R S E          |
//           +-------+-------+---------------+
//
//   * ENC  encoding format
//   * COMP compression type
//   * R    packet is request (client to server) or response (server to client).
//          every request initiates a new session.
//   * S    packet is part of streaming messages.
//   * E    end of session
//
// opaque:
//   * opaque value of 0x80000000 and above is reserved for specific
//     applications.
//   * opaque value from 0xFFFF0000 and above are reserved for protocol.
//   * clients that automatically assigns opaque will have to use values
//     from 1 to 0x7FFFFFFF and start rolling back to 1 after 0x7FFFFFFF.
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

// SetEncoder till set an encoder to encode and decode payload.
func (pkt *TransportPacket) SetEncoder(typ TransportFlag, encoder Encoder) {
	if encoder == nil {
		switch typ {
		case EncodingBinary:
			pkt.encoders[typ] = NewBinaryEncoder()
		}

	} else if encoder != nil {
		pkt.encoders[typ] = encoder

	} else {
		panic("SetEncoder(): encoder is nil")
	}
}

// SetZipper will set a zipper type to compress and decompress payload.
func (pkt *TransportPacket) SetZipper(typ TransportFlag, zipper Compressor) {
	if zipper == nil {
		switch typ {
		case CompressionGzip:
			pkt.zippers[typ] = NewGzipCompression()
		case CompressionLZW:
			pkt.zippers[typ] = NewLZWCompression()
		}

	} else if zipper != nil {
		pkt.zippers[typ] = zipper

	} else {
		panic("SetZipper(): zipper is nil")
	}
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
