package gofast

import "sync"

type Ping struct {
	echo string
}

func (msg *Ping) NewPing(t *Transport, echo string) *Ping {
	val := pingpool.Get()
	msg := val.(*Ping)
	msg.echo = echo
	msg.transport = t
	return msg
}

func (msg *Ping) Id() uint64 {
	return MsgPing
}

func (msg *Ping) Encode(out []byte) int {
	n := arrayStart(out)
	n += valtext2cbor(msg.echo, out[n:], msg.transport.cbor)
	n += breakStop(out[n:])
	return n
}

func (msg *Ping) Decode(in []byte) {
	// echo
	val, n := cbor2value(in, msg.transport.cbor)
	if echo, ok := val.(string); ok {
		msg.echo = echo
	}
	if in[n] == 0xff {
		return
	}
}

func (msg *Ping) Free() {
	pingpool.Put(msg)
}

var pingpool *sync.Pool

func init() {
	pingpool = &sync.Pool{New: func() interface{} { &Ping{} }}
}
