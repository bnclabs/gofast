package gofast

import "sync/atomic"
import "time"
import "reflect"

import "github.com/bnclabs/golog"

const (
	msgStart     uint64 = 0x1000 // reserve start.
	msgPing             = 0x1001 // to ping/echo with peer.
	msgWhoami           = 0x1002 // to supplying/obtaining peer info.
	msgHeartbeat        = 0x1003 // to send/receive heartbeat.
	msgEnd              = 0x100f // reserve end.
)

// handler for whoamiMsg, pingMsg, heartbeatMsg messages.
func (t *Transport) msghandler(stream *Stream, msg BinMessage) StreamCallback {
	switch msg.ID {
	case msgHeartbeat:
		atomic.StoreInt64(&t.aliveat, time.Now().UnixNano())
		atomic.AddUint64(&t.nRxbeats, 1)

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
		typeOfVersion := reflect.ValueOf(t.version).Elem().Type()
		m.version = reflect.New(typeOfVersion).Interface().(Version)
		m.Decode(msg.Data)
		t.peerver.Store(m.version)
		rv := newWhoami(t) // respond back
		if err := stream.Response(rv, true /*flush*/); err != nil {
			log.Errorf("%v response-whoami: %v\n", t.logprefix, err)
		} else {
			atomic.AddInt64(&t.xchngok, 1)
		}

	default:
		log.Errorf("%v message %T:%v not expected\n", t.logprefix, msg, msg)
	}
	return nil
}

func isReservedMsg(id uint64) bool {
	return (msgStart <= id) && (id <= msgEnd)
}
