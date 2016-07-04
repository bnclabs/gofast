package gofast

import "sync/atomic"
import "time"

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
func (t *Transport) msghandler(stream *Stream, msg BinMessage) StreamCallback {
	switch msg.ID {
	case msgHeartbeat:
		atomic.StoreInt64(&t.aliveat, time.Now().UnixNano())
		atomic.AddUint64(&t.n_rxbeats, 1)

	case msgPing:
		var m pingMsg

		m.Decode(msg.Data)
		rv := newPing(m.echo) // respond back
		if err := stream.Response(rv, false /*flush*/); err != nil {
			log.Errorf("%v response-ping: %v\n", t.logprefix, err)
		}

	case msgWhoami:
		var m whoamiMsg

		m.transport = t
		m.Decode(msg.Data)
		t.peerver.Store(m.version)
		rv := newWhoami(t) // respond back
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
