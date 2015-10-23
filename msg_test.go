package gofast

import "testing"
import "strconv"

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

type testMessage struct {
	count uint64
}

func (msg *testMessage) Id() uint64 {
	return msgTest
}

func (msg *testMessage) Encode(out []byte) int {
	n := arrayStart(out)
	n += valuint642cbor(uint64(msg.count), out[n:])
	n += breakStop(out[n:])
	return n
}

func (msg *testMessage) Decode(in []byte) {
	if in[0] != 0x9f {
		return
	}
	ln, _ := cborItemLength(in[1:])
	msg.count = uint64(ln)
}

func (msg *testMessage) String() string {
	return "testMessage"
}

func (msg *testMessage) Repr() string {
	return msg.String() + ":" + strconv.Itoa(int(msg.count))
}
