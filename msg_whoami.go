package gofast

import "sync"
import "strings"
import "strconv"

type Whoami struct {
	transport  *Transport
	Name       string
	Version    interface{}
	Buffersize int
}

func NewWhoami(t *Transport) *Whoami {
	val := whoamipool.Get()
	msg := val.(*Whoami)
	msg.transport = t
	msg.Name = t.name
	msg.Version = t.version.Value()
	msg.Buffersize = t.config["buffersize"].(int)
	return msg
}

func (msg *Whoami) Id() uint64 {
	return MsgWhoami
}

func (msg *Whoami) Encode(out []byte) int {
	n := arrayStart(out)
	n += value2cbor(msg.Name, out[n:])
	n += value2cbor(msg.Version, out[n:])
	n += value2cbor(msg.Buffersize, out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *Whoami) Decode(in []byte) {
	// name
	val, n := cbor2value(in)
	if name, ok := val.(string); ok {
		msg.Name = name
	}
	// version
	val, m := cbor2value(in[n:])
	n += m
	msg.Version = msg.transport.verfunc(val)
	// buffersize
	val, m = cbor2value(in[n:])
	n += m
	if buffersize, ok := val.(uint64); ok {
		msg.Buffersize = int(buffersize)
	}
	if in[n] == 0xff {
		return
	}
}

func (msg *Whoami) String() string {
	return "Whoami"
}

func (msg *Whoami) Repr() string {
	ver := msg.transport.verfunc(msg.Version)
	items := [3]string{
		msg.Name, ver.String(), strconv.Itoa(msg.Buffersize),
	}
	return strings.Join(items[:], ", ")
}

var whoamipool *sync.Pool

func init() {
	whoamipool = &sync.Pool{New: func() interface{} { return &Whoami{} }}
}
