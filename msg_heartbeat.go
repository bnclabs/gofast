package gofast

import "sync"
import "strconv"

type Heartbeat struct {
	count uint64
}

func NewHeartbeat(count uint64) *Heartbeat {
	val := hbpool.Get()
	msg := val.(*Heartbeat)
	msg.count = count
	return msg
}

func (msg *Heartbeat) Id() uint64 {
	return msgHeartbeat
}

func (msg *Heartbeat) Encode(out []byte) int {
	n := arrayStart(out)
	n += value2cbor(msg.count, out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *Heartbeat) Decode(in []byte) {
	// count
	val, n := cbor2value(in)
	if count, ok := val.(uint64); ok {
		msg.count = count
	}
	if in[n] == 0xff {
		return
	}
}

func (msg *Heartbeat) String() string {
	return "Heartbeat"
}

func (msg *Heartbeat) Repr() string {
	return strconv.Itoa(int(msg.count))
}

var hbpool *sync.Pool

func init() {
	hbpool = &sync.Pool{New: func() interface{} { return &Heartbeat{} }}
}
