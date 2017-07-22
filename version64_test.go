package gofast

import "testing"

func TestVersion64(t *testing.T) {
	ver1 := Version64(2)
	if ver1.Size() != 9 {
		t.Errorf("expected 9")
	} else if ref, s := "Version64(2)", ver1.String(); ref != s {
		t.Errorf("expected %v, got %v", ref, s)
	}

	ver2 := Version64(3)
	if ver1.Less(&ver2) == false {
		t.Errorf("expected true")
	} else if ver1.Equal(&ver2) == true {
		t.Errorf("expected false")
	}
	ver2 = Version64(2)
	if ver1.Less(&ver2) == true {
		t.Errorf("expected false")
	} else if ver1.Equal(&ver2) == false {
		t.Errorf("expected true")
	}
	// encoded and decode
	out := ver1.Encode(nil)
	var ver Version64
	if n := ver.Decode(out); n != int64(len(out)) {
		t.Errorf("expected %v, got %v", len(out), n)
	} else if ver.Equal(&ver1) == false {
		t.Errorf("expected %v, got %v", ver1, ver)
	}
}
