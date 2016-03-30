package gofast

// pingMsg is predefined message to ping-pong with remote.
type pingMsg struct {
	echo string
}

func newPing(echo string) *pingMsg {
	return &pingMsg{echo: echo}
}

func (msg *pingMsg) Id() uint64 {
	return msgPing
}

func (msg *pingMsg) Encode(out []byte) int {
	n := arrayStart(out)
	n += valbytes2cbor(str2bytes(msg.echo), out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *pingMsg) Decode(in []byte) {
	n := 0
	if in[n] != 0x9f {
		return
	}
	n += 1
	ln, m := cborItemLength(in[n:])
	n += m
	msg.echo = string(in[n : n+int(ln)])
	return
}

func (msg *pingMsg) String() string {
	return "pingMsg"
}

func (msg *pingMsg) Repr() string {
	return string(msg.echo)
}
