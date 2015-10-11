//  Copyright (c) 2015 Couchbase, Inc.

package gofast

import "testing"
import "fmt"

var _ = fmt.Sprintf("dummy")

func TestBytes2Str(t *testing.T) {
	if bytes2str(nil) != "" {
		t.Errorf("fail bytes2str(nil)")
	}
}

func TestStr2Bytes(t *testing.T) {
	if str2bytes("") != nil {
		t.Errorf(`fail str2bytes("")`)
	}
}

func TestMsgFactory(t *testing.T) {
	ref := &Whoami{}
	wai_factory := msgfactory(ref)
	msg := wai_factory()
	if _, ok := msg.(*Whoami); !ok {
		t.Errorf("expected %v, got %v", ref, msg)
	}
}

func BenchmarkBytes2Str(b *testing.B) {
	bs := []byte("hello world")
	for i := 0; i < b.N; i++ {
		bytes2str(bs)
	}
}

func BenchmarkStr2Bytes(b *testing.B) {
	s := "hello world"
	for i := 0; i < b.N; i++ {
		str2bytes(s)
	}
}

func BenchmarkMsgFactory(b *testing.B) {
	wai_factory := msgfactory(&Whoami{})
	for i := 0; i < b.N; i++ {
		wai_factory()
	}
}
