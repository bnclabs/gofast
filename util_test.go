//  Copyright (c) 2015 Couchbase, Inc.

package gofast

import "testing"
import "reflect"
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

func TestHasString(t *testing.T) {
	if hasString("hello", []string{"hello", "world"}) == false {
		t.Errorf("expected %v, got %v", false, true)
	}
	if hasString("whoami", []string{"hello", "world"}) == true {
		t.Errorf("expected %v, got %v", true, false)
	}
}

func TestCsv2strings(t *testing.T) {
	testcases := [][2]interface{}{
		[2]interface{}{"a ,, b,c", []string{"a", "b", "c"}},
		[2]interface{}{" , a", []string{"a"}},
		[2]interface{}{" a, ", []string{"a"}},
		[2]interface{}{" , ", []string{}},
		[2]interface{}{"", []string{}},
	}
	for _, tcase := range testcases {
		in, ref := tcase[0].(string), tcase[1].([]string)
		if rv := csv2strings(in, []string{}); !reflect.DeepEqual(ref, rv) {
			t.Errorf("expected %v, got %v", ref, rv)
		}
	}
}

func TestCborMap2Golang(t *testing.T) {
	ref := map[string]interface{}{"a": 10, "b": []interface{}{true, false, nil}}
	val := CborMap2golangMap(GolangMap2cborMap(ref))
	if !reflect.DeepEqual(ref, val) {
		t.Errorf("expected %v, got %v", ref, val)
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
