package gofast

import "strconv"

// heartbeatMsg is predefined message, used by transport
// to send periodic heartbeat to remote. Refer to
// SendHeartbeat() method on the transport.
type heartbeatMsg struct {
	count uint64
}

func newHeartbeat(count uint64) *heartbeatMsg {
	return &heartbeatMsg{count: count}
}

func (msg *heartbeatMsg) Id() uint64 {
	return msgHeartbeat
}

func (msg *heartbeatMsg) Encode(out []byte) int {
	n := arrayStart(out)
	n += valuint642cbor(uint64(msg.count), out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *heartbeatMsg) Decode(in []byte) {
	if in[0] != 0x9f {
		return
	}
	ln, _ := cborItemLength(in[1:])
	msg.count = uint64(ln)
}

func (msg *heartbeatMsg) String() string {
	return "heartbeatMsg"
}

func (msg *heartbeatMsg) Repr() string {
	return msg.String() + ":" + strconv.Itoa(int(msg.count))
}
