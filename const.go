package gofast

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

// reserved message-types.
const (
	MtypeBinaryPayload uint16 = 0xF000
	MtypeFlowControl   uint16 = 0xFFFF
)

// reserved opaque values.
const (
	NoOpaque uint32 = 0x00000000
)
