package gofast

import "strconv"
import "encoding/binary"

// heartbeatMsg is predefined message, used by transport
// to send periodic heartbeat to remote. Refer to
// SendHeartbeat() method on the transport.
type heartbeatMsg struct {
	count uint64
}

func newHeartbeat(count uint64) *heartbeatMsg {
	return &heartbeatMsg{count: count}
}

func (msg *heartbeatMsg) ID() uint64 {
	return msgHeartbeat
}

func (msg *heartbeatMsg) Encode(out []byte) []byte {
	out = fixbuffer(out, msg.Size())
	binary.BigEndian.PutUint64(out, msg.count)
	return out[:msg.Size()]
}

func (msg *heartbeatMsg) Decode(in []byte) (n int64) {
	msg.count, n = binary.BigEndian.Uint64(in), n+8
	return n
}

func (msg *heartbeatMsg) Size() int64 {
	return 8
}

func (msg *heartbeatMsg) String() string {
	return "heartbeatMsg"
}

func (msg *heartbeatMsg) Repr() string {
	return msg.String() + ":" + strconv.Itoa(int(msg.count))
}
