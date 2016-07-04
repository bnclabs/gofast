package gofast

import "strings"
import "reflect"
import "strconv"

// whoamiMsg is predefined message to exchange peer information.
type whoamiMsg struct {
	transport  *Transport
	name       string
	version    Version
	buffersize int
	tags       string
}

func newWhoami(t *Transport) *whoamiMsg {
	msg := &whoamiMsg{
		transport:  t,
		name:       t.name,
		version:    t.version,
		buffersize: t.config["buffersize"].(int),
	}
	if tags, ok := t.config["tags"]; ok {
		msg.tags = tags.(string)
	}
	return msg
}

func (msg *whoamiMsg) ID() uint64 {
	return msgWhoami
}

func (msg *whoamiMsg) Encode(out []byte) int {
	n := arrayStart(out)
	n += valbytes2cbor(str2bytes(msg.name), out[n:])
	n += msg.version.Marshal(out[n:])
	n += valuint642cbor(uint64(msg.buffersize), out[n:])
	n += valbytes2cbor(str2bytes(msg.tags), out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *whoamiMsg) Decode(in []byte) {
	n := 0
	if in[n] != 0x9f {
		return
	}
	n += 1
	// name
	ln, m := cborItemLength(in[n:])
	n += m
	msg.name = string(in[n : n+int(ln)])
	n += int(ln)
	// version
	typeOfMsg := reflect.ValueOf(msg.transport.version).Elem().Type()
	msg.version = reflect.New(typeOfMsg).Interface().(Version)
	n += msg.version.Unmarshal(in[n:])
	// buffersize
	ln, m = cborItemLength(in[n:])
	msg.buffersize = int(ln)
	n += m
	// tags
	ln, m = cborItemLength(in[n:])
	n += m
	msg.tags = string(in[n : n+int(ln)])
	n += int(ln)
}

func (msg *whoamiMsg) String() string {
	return "whoamiMsg"
}

func (msg *whoamiMsg) Repr() string {
	items := [2]string{msg.name, strconv.Itoa(msg.buffersize)}
	return strings.Join(items[:], ", ")
}
