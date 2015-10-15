package gofast

import "testing"

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
