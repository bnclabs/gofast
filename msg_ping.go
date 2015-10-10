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
	if in[0] != 0x9f {
		return
	}
	ln, n := cborItemLength(in[1:])
	bs := str2bytes(msg.echo)
	if cap(bs) == 0 {
		bs = make([]byte, len(in))
	}
	bs = append(bs[:0], in[1+n:1+n+ln]...)
	msg.echo = bytes2str(bs[:ln])
	return
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
