package main

import "fmt"
import "encoding/binary"

import "github.com/bnclabs/gofast"

//-- post message for benchmarking

type msgPost struct {
	key   []byte
	value []byte
}

func (msg *msgPost) ID() uint64 {
	return 111
}

func (msg *msgPost) Encode(out []byte) []byte {
	out = fixbuffer(out, msg.Size())
	// encode key
	binary.BigEndian.PutUint64(out, uint64(len(msg.key)))
	n := 8
	n += copy(out[n:], msg.key)
	// encode value
	binary.BigEndian.PutUint64(out, uint64(len(msg.value)))
	n += 8
	n += copy(out[n:], msg.value)
	return out[:n]
}

func (msg *msgPost) Decode(in []byte) int64 {
	// decode key
	ln, n := int64(binary.BigEndian.Uint64(in)), int64(8)
	msg.key = fixbuffer(msg.key, ln)
	n += int64(copy(msg.key, in[n:n+ln]))
	// decode value
	ln, n = int64(binary.BigEndian.Uint64(in[n:])), n+8
	msg.value = fixbuffer(msg.value, ln)
	n += int64(copy(msg.value, in[n:n+ln]))
	return n
}

func (msg *msgPost) Size() int64 {
	return 8 + int64(len(msg.key)) + 8 + int64(len(msg.value))
}

func (msg *msgPost) String() string {
	return "msgPost"
}

//-- version

type testVersion int

func (v *testVersion) Less(ver gofast.Version) bool {
	return (*v) < (*ver.(*testVersion))
}

func (v *testVersion) Equal(ver gofast.Version) bool {
	return (*v) == (*ver.(*testVersion))
}

func (v *testVersion) String() string {
	return fmt.Sprintf("%v", int(*v))
}

func (v *testVersion) Encode(out []byte) []byte {
	out = fixbuffer(out, 32)
	n := valuint642cbor(uint64(*v), out)
	return out[:n]
}

func (v *testVersion) Size() int64 {
	return 9
}

func (v *testVersion) Decode(in []byte) int64 {
	ln, n := cborItemLength(in)
	*v = testVersion(ln)
	return int64(n)
}
