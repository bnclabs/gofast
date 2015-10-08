package gofast

import "sync/atomic"
import "time"

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
	MsgPing uint64 = 0x1
	// MsgWhoami reserved Message Id for supplying/obtaining peer details.
	MsgWhoami = 0x2
	// MsgHeartbeat reserved Message Id to send/receive heartbeat
	MsgHeartbeat = 0x3
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

func (t *Transport) msghandler(stream *Stream, msg Message) chan Message {
	switch m := msg.(type) {
	case *Heartbeat:
		atomic.StoreInt64(&t.aliveat, time.Now().UnixNano())

	case *Ping:
		rv := NewPing(m.echo)
		if err := stream.Response(rv); err != nil {
			log.Errorf("%v response-ping: %v\n", t.logprefix, err)
		}
		t.Free(rv)
		stream.Close()

	case *Whoami:
		rv := NewWhoami(t)
		if err := stream.Response(rv); err != nil {
			log.Errorf("%v response-whoami: %v\n", t.logprefix, err)
		}
		// TODO: make this atomic
		t.peerver = t.verfunc(m.Version)
		t.Free(rv)
		stream.Close()

	default:
		log.Errorf("%v message %T : %v not expected\n", t.logprefix, msg)
	}
	return nil
}
