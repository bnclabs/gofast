package gofast

// Ping is predefined message to ping-pong with remote.
type Ping struct {
	echo string
}

func NewPing(echo string) *Ping {
	return &Ping{echo: echo}
}

func (msg *Ping) Id() uint64 {
	return msgPing
}

func (msg *Ping) Encode(out []byte) int {
	n := arrayStart(out)
	n += valbytes2cbor(str2bytes(msg.echo), out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *Ping) Decode(in []byte) {
	n := 0
	if in[n] != 0x9f {
		return
	}
	n += 1
	ln, m := cborItemLength(in[n:])
	n += m
	msg.echo = string(in[n : n+ln])
	return
}

func (msg *Ping) String() string {
	return "Ping"
}

func (msg *Ping) Repr() string {
	return string(msg.echo)
}
