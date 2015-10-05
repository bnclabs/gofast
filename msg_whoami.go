package gofast

import "sync"

type Whoami struct {
	transport *Transport
	name      string
	version   interface{}
	buflen    int
	heartbeat uint64 // in millisecond
}

func (msg *Whoami) NewWhoami(t *Transport) *Whoami {
	val := whoamipool.Get()
	msg := val.(*Whoami)
	msg.name = t.name
	msg.version = t.version.Value()
	msg.buflen = t.buflen
	msg.heartbeat = t.heartbeat
	msg.transport = t
	return msg
}

func (msg *Whoami) Id() uint64 {
	return MsgWhoami
}

func (msg *Whoami) Encode(out []byte) int {
	n := arrayStart(out)
	n += valtext2cbor(msg.name, out[n:], msg.transport.cbor)
	n += value2cbor(msg.version, out[n:], msg.transport.cbor)
	n += valtext2cbor(msg.buflen, out[n:], msg.transport.cbor)
	n += valtext2cbor(msg.heartbeat, out[n:], msg.transport.cbor)
	n += breakStop(out[n:])
	return n
}

func (msg *Whoami) Decode(in []byte) {
	// name
	val, n := cbor2value(in, msg.transport.cbor)
	if name, ok := val.(string); ok {
		msg.name = name
	}
	// version
	val, m = cbor2value(in[n:], msg.transport.cbor)
	n += m
	msg.version = msg.transport.verfunc(val)
	// buflen
	val, m = cbor2value(in[n:], msg.transport.cbor)
	n += m
	if buflen, ok := val.(uint64); ok {
		msg.buflen = buflen
	}
	// heartbeat
	val, m = cbor2value(in[n:], msg.transport.cbor)
	n += m
	if heartbeat, ok := val.(uint64); ok {
		msg.heartbeat = heartbeat
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
	whoamipool = &sync.Pool{New: func() interface{} { &Whoami{} }}
}
