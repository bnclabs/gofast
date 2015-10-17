package gofast

import "sync"
import "strings"
import "reflect"
import "strconv"

// Whoami is predefined message to exchange peer information.
type Whoami struct {
	transport  *Transport
	name       []byte
	version    Version
	buffersize int
	tags       []byte
}

func NewWhoami(t *Transport) *Whoami {
	val := whoamipool.Get()
	msg := val.(*Whoami)
	msg.transport = t
	if msg.name == nil {
		msg.name = make([]byte, 16)
	}
	msg.name = append(msg.name[:0], str2bytes(t.name)...)
	msg.version = t.version
	msg.buffersize = t.config["buffersize"].(int)
	if msg.tags == nil {
		msg.tags = make([]byte, 0, 16)
	}
	if tags, ok := t.config["tags"]; ok {
		msg.tags = append(msg.tags[:0], str2bytes(tags.(string))...)
	}
	return msg
}

func (msg *Whoami) Id() uint64 {
	return msgWhoami
}

func (msg *Whoami) Encode(out []byte) int {
	n := arrayStart(out)
	n += valbytes2cbor(msg.name, out[n:])
	n += msg.version.Marshal(out[n:])
	n += valuint642cbor(uint64(msg.buffersize), out[n:])
	n += valbytes2cbor(msg.tags, out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *Whoami) Decode(in []byte) {
	n := 0
	if in[n] != 0x9f {
		return
	}
	n += 1
	// name
	ln, m := cborItemLength(in[n:])
	n += m
	if msg.name == nil {
		msg.name = make([]byte, ln)
	}
	msg.name = append(msg.name[:0], in[n:n+ln]...)
	n += ln
	// version
	if msg.version == nil {
		typeOfMsg := reflect.ValueOf(msg.transport.version).Elem().Type()
		msg.version = reflect.New(typeOfMsg).Interface().(Version)
	}
	n += msg.version.Unmarshal(in[n:])
	// buffersize
	ln, m = cborItemLength(in[n:])
	msg.buffersize = ln
	n += m
	// tags
	ln, m = cborItemLength(in[n:])
	n += m
	if msg.tags == nil {
		msg.tags = make([]byte, ln)
	}
	msg.tags = append(msg.tags[:0], in[n:n+ln]...)
	n += ln
}

func (msg *Whoami) String() string {
	return "Whoami"
}

func (msg *Whoami) Repr() string {
	items := [2]string{string(msg.name), strconv.Itoa(msg.buffersize)}
	return strings.Join(items[:], ", ")
}

var whoamipool *sync.Pool

func init() {
	whoamipool = &sync.Pool{New: func() interface{} { return &Whoami{} }}
}
