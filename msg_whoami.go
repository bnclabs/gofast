package gofast

import "sync"
import "strings"
import "reflect"
import "strconv"

// Whoami is predefined message to exchange peer information.
type Whoami struct {
	transport  *Transport
	name       string
	version    Version
	buffersize int
	tags       string
}

func NewWhoami(t *Transport) *Whoami {
	val := whoamipool.Get()
	msg := val.(*Whoami)
	msg.transport = t
	msg.name = t.name
	msg.version = t.version
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
	n += valtext2cbor(msg.name, out[n:])
	n += msg.version.Marshal(out[n:])
	n += valuint642cbor(uint64(msg.buffersize), out[n:])
	n += valtext2cbor(msg.tags, out[n:])
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
	bs := str2bytes(msg.name)
	if cap(bs) == 0 {
		bs = make([]byte, len(in))
	}
	bs = append(bs[:0], in[n:n+ln]...)
	msg.name = bytes2str(bs)
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
	bs = str2bytes(msg.tags)
	if cap(bs) == 0 {
		bs = make([]byte, len(in))
	}
	bs = append(bs[:0], in[n:n+ln]...)
	msg.tags = bytes2str(bs)
	n += ln
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
