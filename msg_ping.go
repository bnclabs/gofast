package gofast

import "sync"

type Ping struct {
	echo string
}

func NewPing(echo string) *Ping {
	val := pingpool.Get()
	msg := val.(*Ping)
	msg.echo = echo
	return msg
}

func (msg *Ping) Id() uint64 {
	return msgPing
}

func (msg *Ping) Encode(out []byte) int {
	n := arrayStart(out)
	n += valtext2cbor(msg.echo, out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *Ping) Decode(in []byte) {
	// echo
	val, n := cbor2value(in)
	if echo, ok := val.(string); ok {
		msg.echo = echo
	}
	if in[n] == 0xff {
		return
	}
}

func (msg *Ping) String() string {
	return "Ping"
}

func (msg *Ping) Repr() string {
	return msg.echo
}

var pingpool *sync.Pool

func init() {
	pingpool = &sync.Pool{New: func() interface{} { return &Ping{} }}
}
