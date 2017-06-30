package gofast

import "fmt"
import "testing"
import "strconv"
import "encoding/binary"

func TestIsReservedMsg(t *testing.T) {
	if isReservedMsg(msgStart) == false {
		t.Errorf("failed for msgStart")
	} else if isReservedMsg(msgEnd) == false {
		t.Errorf("failed for msgEnd")
	} else if isReservedMsg(msgPing) == false {
		t.Errorf("failed for msgPing")
	} else if isReservedMsg(msgWhoami) == false {
		t.Errorf("failed for msgWhoami")
	} else if isReservedMsg(msgHeartbeat) == false {
		t.Errorf("failed for msgHeartbeat")
	}
}

const msgTest = msgEnd + 1
const msgEmpty = msgTest + 1
const msgOnebyte = msgTest + 1
const msgLarge = msgTest + 1

//-- test message

type testMessage struct {
	count uint64
}

func (msg *testMessage) ID() uint64 {
	return msgTest
}

func (msg *testMessage) Encode(out []byte) []byte {
	out = fixbuffer(out, msg.Size())
	binary.BigEndian.PutUint64(out, msg.count)
	return out[:msg.Size()]
}

func (msg *testMessage) Decode(in []byte) (n int64) {
	msg.count = binary.BigEndian.Uint64(in)
	return msg.Size()
}

func (msg *testMessage) Size() int64 {
	return 8
}

func (msg *testMessage) String() string {
	return "testMessage"
}

func (msg *testMessage) Repr() string {
	return msg.String() + ":" + strconv.Itoa(int(msg.count))
}

//-- empty message

type emptyMessage struct {
}

func (msg *emptyMessage) ID() uint64 {
	return msgEmpty
}

func (msg *emptyMessage) Encode(out []byte) []byte {
	return out[:0]
}

func (msg *emptyMessage) Decode(in []byte) int64 {
	return 0
}

func (msg *emptyMessage) Size() int64 {
	return 0
}

func (msg *emptyMessage) String() string {
	return "emptyMessage"
}

func (msg *emptyMessage) Repr() string {
	return msg.String()
}

//-- onebyte message

type onebyteMessage struct {
	field byte
}

func (msg *onebyteMessage) ID() uint64 {
	return msgOnebyte
}

func (msg *onebyteMessage) Encode(out []byte) []byte {
	out[0] = msg.field
	return out[:1]
}

func (msg *onebyteMessage) Decode(in []byte) int64 {
	msg.field = in[0]
	return 1
}

func (msg *onebyteMessage) Size() int64 {
	return 1
}

func (msg *onebyteMessage) String() string {
	return "onebyteMessage"
}

func (msg *onebyteMessage) Repr() string {
	return fmt.Sprintf("onebyteMessage.%v", msg.field)
}

//-- large message

type largeMessage struct {
	data [65 * 1024]byte
}

func (msg *largeMessage) ID() uint64 {
	return msgLarge
}

func (msg *largeMessage) Encode(out []byte) []byte {
	out = fixbuffer(out, msg.Size())
	binary.BigEndian.PutUint64(out, uint64(len(msg.data)))
	n := 8
	n += copy(out, msg.data[:])
	return out[:n]
}

func (msg *largeMessage) Decode(in []byte) int64 {
	ln, n := int(binary.BigEndian.Uint64(in)), 8
	n += copy(msg.data[:], in[n:n+ln])
	return int64(n)
}

func (msg *largeMessage) Size() int64 {
	return 8 + int64(len(msg.data))
}

func (msg *largeMessage) String() string {
	return "largeMessage"
}

func (msg *largeMessage) Repr() string {
	return msg.String()
}
