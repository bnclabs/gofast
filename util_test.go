//  Copyright (c) 2015 Couchbase, Inc.

package gofast

import "testing"
import "encoding/json"
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

func TestCborMap2Golang(t *testing.T) {
	var value interface{}

	ref := `{"a":10,"b":[true,false,null]}`
	cborcode := make([]byte, 1024)
	config := NewDefaultConfig()
	json.Unmarshal([]byte(ref), &value)
	n := config.ValueToCbor(GolangMap2cborMap(value), cborcode)
	value1, _ := config.CborToValue(cborcode[:n])
	data, err := json.Marshal(CborMap2golangMap(value1))
	if err != nil {
		t.Fatalf("json parsing: %v\n	%v", value1, err)
	}
	if s := string(data); s != ref {
		t.Errorf("expected %q, got %q", ref, s)
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
