package gofast

import "errors"

// ErrorInvalidMtype detects invalid message type in transport
// frame.
var ErrorInvalidMtype = errors.New("gofast.invalidMtype")

// ErrorPacketWrite is error writing packet on the wire.
var ErrorPacketWrite = errors.New("gofast.packetWrite")

// ErrorPacketRead is error reading packet on the wire.
var ErrorPacketRead = errors.New("gofast.packetRead")

// ErrorPacketOverflow is input packet overflows maximum
// configured packet size.
var ErrorPacketOverflow = errors.New("gofast.packetOverflow")

// ErrorEncoderUnknown for unknown encoder.
var ErrorEncoderUnknown = errors.New("gofast.encoderUnknown")

// ErrorZipperUnknown for unknown compression.
var ErrorZipperUnknown = errors.New("gofast.zipperUnknown")

// ErrorClosed by client APIs when client instance is already
// closed, helpful when there is a race between API and Close()
var ErrorClosed = errors.New("gofast.clientClosed")

// ErrorDuplicateRequest means a request is already in flight with
// supplied opaque.
var ErrorDuplicateRequest = errors.New("gofast.duplicateRequest")

// ErrorBadRequest is returned for malformed request.
var ErrorBadRequest = errors.New("gofast.badRequest")

// ErrorBadPayload is returned for malformed payload type.
var ErrorBadPayload = errors.New("gofast.badPayload")

// ErrorUnknownStream is returned by client.
var ErrorUnknownStream = errors.New("gofast.unknownStream")

// reserved message-types.
const (
	MtypeBinaryPayload uint16 = 0xF000
	MtypeFlowControl   uint16 = 0xFFFE
	MtypeEmpty         uint16 = 0xFFFF
)

// reserved opaque values.
const (
	NoOpaque uint32 = 0x00000000
)
