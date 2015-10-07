package gofast

import "testing"

import "io/ioutil"
import "fmt"

var _ = fmt.Sprintf("dummy")

func TestTagLzw(t *testing.T) {
	tag, enc, dec := make_lzw(nil, nil)
	if tag != tagLzw {
		t.Errorf("expected %v, got %v", tagLzw, tag)
	}
	// test with valid input
	ref := "hello world"
	in, out := make([]byte, 1024*1024), make([]byte, 1024*1024)
	n := enc([]byte(ref), in)
	fmt.Println(n, in[:n])
	m := dec(in[:n], out)
	if s := string(out[:m]); s != ref {
		t.Errorf("expected %v, got %v", ref, s)
	}
	// test with empty input for encoder
	if n = enc([]byte{}, in); n != 0 {
		t.Errorf("expected %v, got %v", 0, n)
	}
	// test with empty input for decoder
	if n = dec([]byte{}, in); n != 0 {
		t.Errorf("expected %v, got %v", 0, n)
	}
}

func BenchmarkLzwEnc1KFast(b *testing.B) {
	_, enc, _ := make_lzw(nil, nil)
	s, err := ioutil.ReadFile("testdata/1k.json")
	if err != nil {
		panic(err)
	}
	out := make([]byte, 1024*1024)
	b.ResetTimer()
	var n int
	for i := 0; i < b.N; i++ {
		n = enc(s, out)
	}
	//fmt.Println(n)
	b.SetBytes(int64(n))
}

func BenchmarkLzwDec1KFast(b *testing.B) {
	s, err := ioutil.ReadFile("testdata/1k.json")
	if err != nil {
		panic(err)
	}
	_, enc, dec := make_lzw(nil, nil)
	in, out := make([]byte, 1024*1024), make([]byte, 1024*1024)
	n := enc(s, in)
	b.ResetTimer()
	var m int
	for i := 0; i < b.N; i++ {
		m = dec(in[:n], out)
	}
	// fmt.Println(m)
	b.SetBytes(int64(m))
}

func BenchmarkLzwEnc10KSmall(b *testing.B) {
	_, enc, _ := make_lzw(nil, nil)
	s, err := ioutil.ReadFile("testdata/1k.json")
	if err != nil {
		panic(err)
	}
	in, out := make([]byte, 1024*1024*10), make([]byte, 1024*1024*10)
	n := 0
	for i := 0; i < 10; i++ {
		n += copy(in, s)
	}
	b.ResetTimer()
	var m int
	for i := 0; i < b.N; i++ {
		m = enc(in[:n], out)
	}
	// fmt.Println(m)
	b.SetBytes(int64(m))
}

func BenchmarkLzwDec10KSmall(b *testing.B) {
	_, enc, dec := make_lzw(nil, nil)
	s, err := ioutil.ReadFile("testdata/1k.json")
	if err != nil {
		panic(err)
	}
	in, out := make([]byte, 1024*1024*10), make([]byte, 1024*1024*10)
	n := 0
	for i := 0; i < 10; i++ {
		n += copy(in, s)
	}
	p := enc(in[:n], out)
	b.ResetTimer()
	var m int
	for i := 0; i < b.N; i++ {
		m = dec(out[:p], in)
	}
	// fmt.Println(m)
	b.SetBytes(int64(m))
}
