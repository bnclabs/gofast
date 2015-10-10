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
	msgStart     uint64 = 0x0 // reserve start.
	msgPing             = 0x1 // to ping/echo with peer.
	msgWhoami           = 0x2 // to supplying/obtaining peer info.
	msgHeartbeat        = 0x3 // to send/receive heartbeat.
	msgEnd              = 0xf // reserve end.
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
		rv := NewPing(m.echo) // respond back
		defer t.Free(rv)
		defer stream.Close()
		if err := stream.Response(rv); err != nil {
			log.Errorf("%v response-ping: %v\n", t.logprefix, err)
		}

	case *Whoami:
		t.peerver = t.verfunc(m.version) // TODO: make this atomic
		rv := NewWhoami(t)               // respond back
		defer t.Free(rv)
		defer stream.Close()
		if err := stream.Response(rv); err != nil {
			log.Errorf("%v response-whoami: %v\n", t.logprefix, err)
		}

	default:
		log.Errorf("%v message %T : %v not expected\n", t.logprefix, msg)
	}
	return nil
}

func isReservedMsg(id uint64) bool {
	return (msgStart <= id) && (id <= msgEnd)
}
