package gofast

import "sync"
import "strings"
import "strconv"

type Whoami struct {
	transport  *Transport
	name       string
	version    interface{}
	buffersize int
	tags       string
}

func NewWhoami(t *Transport) *Whoami {
	val := whoamipool.Get()
	msg := val.(*Whoami)
	msg.transport = t
	msg.name = t.name
	msg.version = t.version.Value()
	msg.buffersize = t.config["buffersize"].(int)
	msg.tags = ""
	if tags, ok := t.config["tags"]; ok {
		msg.tags = tags.(string)
	}
	return msg
}

func (msg *Whoami) Id() uint64 {
	return msgWhoami
}

func (msg *Whoami) Encode(out []byte) int {
	n := arrayStart(out)
	n += value2cbor(msg.name, out[n:])
	n += value2cbor(msg.version, out[n:])
	n += value2cbor(msg.buffersize, out[n:])
	n += value2cbor(msg.tags, out[n:])
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
	// buffersize
	val, m = cbor2value(in[n:])
	n += m
	if buffersize, ok := val.(uint64); ok {
		msg.buffersize = int(buffersize)
	}
	// tags
	val, m = cbor2value(in[n:])
	n += m
	if tags, ok := val.(string); ok {
		msg.tags = tags
	}
	if in[n] == 0xff {
		return
	}
}

func (msg *Whoami) String() string {
	return "Whoami"
}

func (msg *Whoami) Repr() string {
	ver := msg.transport.verfunc(msg.version)
	items := [3]string{
		msg.name, ver.String(), strconv.Itoa(msg.buffersize),
	}
	return strings.Join(items[:], ", ")
}

var whoamipool *sync.Pool

func init() {
	whoamipool = &sync.Pool{New: func() interface{} { return &Whoami{} }}
}
