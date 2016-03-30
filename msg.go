package gofast

import "sync/atomic"
import "time"

// Message interface, implemented by all messages exchanged via
// gofast-transport.
type Message interface {
	// Id return a unique message identifier.
	Id() uint64

	// Encode message to binary blob.
	Encode(out []byte) int

	// Decode this message from a binary blob.
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

// Version interface to exchange and manage transport's version.
type Version interface {
	// Less than supplied version.
	Less(ver Version) bool

	// Equal to the supplied version.
	Equal(ver Version) bool

	// String representation of version, for logging.
	String() string

	// Marshal version into array of bytes.
	Marshal(out []byte) (n int)

	// Unmarshal array of bytes to version object.
	Unmarshal(out []byte) (n int)
}

// handler for whoamiMsg, pingMsg, heartbeatMsg messages.
func (t *Transport) msghandler(stream *Stream, msg Message) chan Message {
	defer t.Free(msg)

	switch m := msg.(type) {
	case *heartbeatMsg:
		atomic.StoreInt64(&t.aliveat, time.Now().UnixNano())
		atomic.AddUint64(&t.n_rxbeats, 1)

	case *pingMsg:
		rv := newPing(m.echo) // respond back
		if err := stream.Response(rv, false /*flush*/); err != nil {
			log.Errorf("%v response-ping: %v\n", t.logprefix, err)
		}

	case *whoamiMsg:
		t.peerver = m.version // TODO: make this atomic
		rv := newWhoami(t)    // respond back
		if err := stream.Response(rv, true /*flush*/); err != nil {
			log.Errorf("%v response-whoami: %v\n", t.logprefix, err)
		} else {
			atomic.AddInt64(&t.xchngok, 1)
		}

	default:
		log.Errorf("%v message %T : %v not expected\n", t.logprefix, msg)
	}
	return nil
}

func isReservedMsg(id uint64) bool {
	return (msgStart <= id) && (id <= msgEnd)
}
