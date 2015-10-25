package gofast

import "strconv"

// Heartbeat is predefined message, used by transport
// to send periodic heartbeat to remote. Refer to
// SendHeartbeat() method on the transport.
type Heartbeat struct {
	count uint64
}

func NewHeartbeat(count uint64) *Heartbeat {
	return &Heartbeat{count: count}
}

func (msg *Heartbeat) Id() uint64 {
	return msgHeartbeat
}

func (msg *Heartbeat) Encode(out []byte) int {
	n := arrayStart(out)
	n += valuint642cbor(uint64(msg.count), out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *Heartbeat) Decode(in []byte) {
	if in[0] != 0x9f {
		return
	}
	ln, _ := cborItemLength(in[1:])
	msg.count = uint64(ln)
}

func (msg *Heartbeat) String() string {
	return "Heartbeat"
}

func (msg *Heartbeat) Repr() string {
	return msg.String() + ":" + strconv.Itoa(int(msg.count))
}
