package gofast

import "sync"

// Ping is predefined message to ping-pong with remote.
type Ping struct {
	echo []byte
}

func NewPing(echo string) *Ping {
	val := pingpool.Get()
	msg := val.(*Ping)
	if msg.echo == nil {
		msg.echo = []byte(echo)
	} else {
		msg.echo = append(msg.echo[:0], str2bytes(echo)...)
	}
	return msg
}

func (msg *Ping) Id() uint64 {
	return msgPing
}

func (msg *Ping) Encode(out []byte) int {
	n := arrayStart(out)
	n += valbytes2cbor(msg.echo, out[n:])
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
	if msg.echo == nil {
		msg.echo = make([]byte, ln)
	}
	msg.echo = append(msg.echo[:0], in[n:n+ln]...)
	return
}

func (msg *Ping) String() string {
	return "Ping"
}

func (msg *Ping) Repr() string {
	return string(msg.echo)
}

var pingpool *sync.Pool

func init() {
	pingpool = &sync.Pool{New: func() interface{} { return &Ping{} }}
}
