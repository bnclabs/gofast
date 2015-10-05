package gofast

import "sync"

type Whoami struct {
	transport *Transport
	name      string
	version   interface{}
	buflen    int
}

func NewWhoami(t *Transport) *Whoami {
	val := whoamipool.Get()
	msg := val.(*Whoami)
	msg.transport = t
	msg.name = t.name
	msg.version = t.version.Value()
	msg.buflen = t.config["buflen"].(int)
	return msg
}

func (msg *Whoami) Id() uint64 {
	return MsgWhoami
}

func (msg *Whoami) Encode(out []byte) int {
	n := arrayStart(out)
	n += value2cbor(msg.name, out[n:])
	n += value2cbor(msg.version, out[n:])
	n += value2cbor(msg.buflen, out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *Whoami) Decode(in []byte) {
	// name
	val, n := cbor2value(in)
	if name, ok := val.(string); ok {
		msg.name = name
	}
	// version
	val, m := cbor2value(in[n:])
	n += m
	msg.version = msg.transport.verfunc(val)
	// buflen
	val, m = cbor2value(in[n:])
	n += m
	if buflen, ok := val.(uint64); ok {
		msg.buflen = int(buflen)
	}
	if in[n] == 0xff {
		return
	}
}

func (msg *Whoami) Free() {
	whoamipool.Put(msg)
}

var whoamipool *sync.Pool

func init() {
	whoamipool = &sync.Pool{New: func() interface{} { return &Whoami{} }}
}
