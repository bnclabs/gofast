package gofast

import "encoding/binary"

// pingMsg is predefined message to ping-pong with remote.
type pingMsg struct {
	echo string
}

func newPing(echo string) *pingMsg {
	return &pingMsg{echo: echo}
}

func (msg *pingMsg) ID() uint64 {
	return msgPing
}

func (msg *pingMsg) Encode(out []byte) []byte {
	out = fixbuffer(out, msg.Size())
	binary.BigEndian.PutUint16(out, uint16(len(msg.echo)))
	n := 2
	n += copy(out[n:], msg.echo)
	return out[:n]
}

func (msg *pingMsg) Decode(in []byte) int64 {
	ln, n := int64(binary.BigEndian.Uint16(in)), int64(2)
	msg.echo, n = string(in[n:n+ln]), n+ln
	return n
}

func (msg *pingMsg) Size() int64 {
	return 2 + int64(len(msg.echo))
}

func (msg *pingMsg) String() string {
	return "pingMsg"
}

func (msg *pingMsg) Repr() string {
	return string(msg.echo)
}
