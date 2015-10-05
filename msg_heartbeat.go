package gofast

import "sync"

type Heartbeat struct {
	transport *Transport
	count     uint64
}

func (msg *Heartbeat) NewHeartbeat(t *Transport, count uint64) *Heartbeat {
	val := hbpool.Get()
	msg := val.(*Heartbeat)
	msg.count = count
	msg.transport = t
	return msg
}

func (msg *Heartbeat) Id() uint64 {
	return MsgHeartbeat
}

func (msg *Heartbeat) Encode(out []byte) int {
	n := arrayStart(out)
	n += valtext2cbor(msg.count, out[n:], msg.transport.cbor)
	n += breakStop(out[n:])
	return n
}

func (msg *Heartbeat) Decode(in []byte) {
	// count
	val, n := cbor2value(in, msg.transport.cbor)
	if count, ok := val.(uint64); ok {
		msg.count = count
	}
	if in[n] == 0xff {
		return
	}
}

func (msg *Heartbeat) Free() {
	hbpool.Put(msg)
}

var hbpool *sync.Pool

func init() {
	hbpool = &sync.Pool{New: func() interface{} { &Heartbeat{} }}
}
