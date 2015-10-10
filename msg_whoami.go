package gofast

import "sync"
import "strings"
import "strconv"

type Whoami struct {
	name       string
	version    interface{}
	buffersize int
	tags       string
}

func NewWhoami(t *Transport) *Whoami {
	val := whoamipool.Get()
	msg := val.(*Whoami)
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
	val, _ := cbor2value(in)
	if items, ok := val.([]interface{}); ok {
		msg.name = items[0].(string)
		msg.version = items[1]
		msg.buffersize = int(items[2].(uint64))
		msg.tags = items[3].(string)

	} else {
		log.Errorf("Whoami{}.Decode() invalid i/p\n")
	}
}

func (msg *Whoami) String() string {
	return "Whoami"
}

func (msg *Whoami) Repr() string {
	items := [2]string{msg.name, strconv.Itoa(msg.buffersize)}
	return strings.Join(items[:], ", ")
}

var whoamipool *sync.Pool

func init() {
	whoamipool = &sync.Pool{New: func() interface{} { return &Whoami{} }}
}
