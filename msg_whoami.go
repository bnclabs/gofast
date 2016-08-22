package gofast

import "fmt"
import "encoding/binary"

// whoamiMsg is predefined message to exchange peer information.
type whoamiMsg struct {
	transport  *Transport
	name       string
	version    Version
	buffersize uint64
	tags       string
}

func newWhoami(t *Transport) *whoamiMsg {
	msg := &whoamiMsg{
		transport:  t,
		name:       t.name,
		version:    t.version,
		buffersize: t.buffersize,
	}
	msg.tags = t.settings.String("tags")
	return msg
}

func (msg *whoamiMsg) ID() uint64 {
	return msgWhoami
}

func (msg *whoamiMsg) Encode(out []byte) []byte {
	// TODO: there seem to be unexpected memory allocation happening here.
	out = fixbuffer(out, msg.Size())

	var scratch [256]byte
	n := 0
	out[n], n = byte(len(msg.name)), n+1
	n += copy(out[n:], msg.name)
	scratchout := msg.version.Encode(scratch[:])
	n += copy(out[n:], scratchout)
	binary.BigEndian.PutUint64(out[n:], msg.buffersize)
	n += 8
	binary.BigEndian.PutUint16(out[n:], uint16(len(msg.tags)))
	n += 2
	n += copy(out[n:], msg.tags)
	return out[:n]
}

func (msg *whoamiMsg) Decode(in []byte) int64 {
	ln, n := int64(in[0]), int64(1)
	msg.name, n = string(in[n:n+ln]), n+ln
	n += msg.version.Decode(in[n:])
	msg.buffersize, n = binary.BigEndian.Uint64(in[n:]), n+8
	ln, n = int64(binary.BigEndian.Uint16(in[n:])), n+2
	msg.tags, n = string(in[n:n+ln]), n+ln
	return n
}

func (msg *whoamiMsg) Size() int64 {
	return 1 + int64(len(msg.name)) +
		msg.version.Size() + 8 + 2 + int64(len(msg.tags))
}

func (msg *whoamiMsg) String() string {
	return "whoamiMsg"
}

func (msg *whoamiMsg) Repr() string {
	return fmt.Sprintf("%s,%v", msg.name, msg.buffersize)
}
