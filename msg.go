package gofast

// Message interface specification for all messages are
// exchanged via gofast.
type Message interface {
	// Id return a unique message identifier.
	Id() uint64
	// Encode message to binary blob.
	Encode(out []byte) int
	// Decode this messae from a binary blob.
	Decode(in []byte)
	// String representation of this message, used for logging.
	String() string
}

const (
	// MsgPing reserved Message Id for ping/echo with peer.
	MsgPing = 0xFFFFFFFFFFFFFFFF
	// MsgWhoami reserved Message Id for supplying/obtaining peer details.
	MsgWhoami uint64 = 0xFFFFFFFFFFFFFFFE
	// MsgHeartbeat reserved Message Id to send/receive heartbeat
	MsgHeartbeat = 0x1
)

// Version interface for all messages that are exchanged via gofast.
type Version interface {
	// Less than supplied version.
	Less(ver Version) bool
	// Equal to the supplied version.
	Equal(ver Version) bool
	// String representation of message, used for logging.
	String() string
	// Convert to basic data-type that can be encoded via CBOR.
	Value() interface{}
}
