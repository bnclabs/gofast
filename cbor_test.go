package gofast

import "testing"

func TestCborHdr(t *testing.T) {
	brkstp := cborHdr(cborType7, cborItemBreak)
	if major := cborMajor(brkstp); major != cborType7 {
		t.Errorf("expected %v, got %v", cborType7, major)
	} else if info := cborInfo(brkstp); info != cborItemBreak {
		t.Errorf("expected %v, got %v", cborType7, info)
	}
}

func TestValuint642cbor(t *testing.T) {
	out := make([]byte, 9)
	if n := valuint642cbor(4294967296, out); n != 9 {
		t.Errorf("expected %v, got %v", 9, n)
	} else if ln, _ := cborItemLength(out[:n]); ln != 4294967296 {
		t.Errorf("expected %v, got %v", 4294967296, ln)
	}
}

func TestValtext2cbor(t *testing.T) {
	ref, out := "hello world", make([]byte, 64)
	if n := valtext2cbor(ref, out); n != 12 {
		t.Errorf("expected %v, got %v", 12, n)
	} else if ln, m := cborItemLength(out[:n]); ln != 11 {
		t.Errorf("expected %v, got %v", 11, ln)
	} else if s := string(out[m:n]); s != ref {
		t.Errorf("expected %v, got %v", ref, s)
	}
}
