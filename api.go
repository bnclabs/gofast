package gofast

// Streamer interface to send/receive bidirectional streams of several
// messages.
type Streamer interface {
	// Response to a request, to batch the response pass flush as false.
	Response(msg Message, flush bool) error

	// Stream a single message, to batch the message pass flush as false.
	Stream(msg Message, flush bool) error

	// Close this stream.
	Close() error
}

// BinMessage is a tuple of {id, encodedmsg-slice}
type BinMessage struct {
	ID   uint64
	Data []byte
}

// Message interface, implemented by all messages exchanged via
// gofast-transport.
type Message interface {
	// ID return a unique message identifier.
	ID() uint64

	// Encode message to binary blob.
	Encode(out []byte) []byte

	// Decode this message from a binary blob.
	Decode(in []byte) (n int64)

	// Size is maximum memory foot-print needed for encoding this message.
	Size() int64

	// String representation of this message, used for logging.
	String() string
}

// Version interface to exchange and manage transport's version.
type Version interface {
	// Less than supplied version.
	Less(ver Version) bool

	// Equal to the supplied version.
	Equal(ver Version) bool

	// Encode version into array of bytes.
	Encode(out []byte) []byte

	// Decode array of bytes to version object.
	Decode(in []byte) (n int64)

	// Size return memory foot-print encoded version
	Size() int64

	// String representation of version, for logging.
	String() string
}
