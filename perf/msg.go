package main

import "fmt"
import "encoding/binary"

import "github.com/prataprc/gofast"

//-- post message for benchmarking

type msgPost struct {
	data []byte
}

func newMsgPost(data []byte) *msgPost {
	return &msgPost{data: data}
}

func (msg *msgPost) ID() uint64 {
	return 111
}

func (msg *msgPost) Encode(out []byte) []byte {
	out = fixbuffer(out, msg.Size())
	binary.BigEndian.PutUint64(out, uint64(len(msg.data)))
	n := 8
	n += copy(out[n:], msg.data)
	return out[:n]
}

func (msg *msgPost) Decode(in []byte) int64 {
	ln, n := int64(binary.BigEndian.Uint64(in)), int64(8)
	msg.data = fixbuffer(msg.data, ln)
	n += int64(copy(msg.data, in[n:n+ln]))
	return n
}

func (msg *msgPost) Size() int64 {
	return 8 + int64(len(msg.data))
}

func (msg *msgPost) String() string {
	return "msgPost"
}

//-- reqsp message for benchmarking

type msgReqsp struct {
	data []byte
}

func (msg *msgReqsp) ID() uint64 {
	return 112
}

func (msg *msgReqsp) Encode(out []byte) []byte {
	out = fixbuffer(out, msg.Size())
	binary.BigEndian.PutUint64(out, uint64(len(msg.data)))
	n := 8
	n += copy(out[n:], msg.data)
	return out[:n]
}

func (msg *msgReqsp) Decode(in []byte) int64 {
	ln, n := int64(binary.BigEndian.Uint64(in)), int64(8)
	msg.data = fixbuffer(msg.data, ln)
	n += int64(copy(msg.data, in[n:n+ln]))
	return n
}

func (msg *msgReqsp) Size() int64 {
	return 8 + int64(len(msg.data))
}
func (msg *msgReqsp) String() string {
	return "msgReqsp"
}

//-- streamrx message for benchmarking

type msgStreamRx struct {
	data []byte
}

func (msg *msgStreamRx) ID() uint64 {
	return 113
}

func (msg *msgStreamRx) Encode(out []byte) []byte {
	out = fixbuffer(out, msg.Size())
	binary.BigEndian.PutUint64(out, uint64(len(msg.data)))
	n := 8
	n += copy(out[n:], msg.data)
	return out[:n]
}

func (msg *msgStreamRx) Decode(in []byte) int64 {
	ln, n := int64(binary.BigEndian.Uint64(in)), int64(8)
	msg.data = fixbuffer(msg.data, ln)
	n += int64(copy(msg.data, in[n:n+ln]))
	return n
}

func (msg *msgStreamRx) Size() int64 {
	return 8 + int64(len(msg.data))
}

func (msg *msgStreamRx) String() string {
	return "msgStreamRx"
}

//-- streamtx message for benchmarking

type msgStreamTx struct {
	data []byte
}

func (msg *msgStreamTx) ID() uint64 {
	return 114
}

func (msg *msgStreamTx) Encode(out []byte) []byte {
	out = fixbuffer(out, msg.Size())
	binary.BigEndian.PutUint64(out, uint64(len(msg.data)))
	n := 8
	n += copy(out[n:], msg.data)
	return out[:n]
}

func (msg *msgStreamTx) Decode(in []byte) int64 {
	ln, n := int64(binary.BigEndian.Uint64(in)), int64(8)
	msg.data = fixbuffer(msg.data, ln)
	n += int64(copy(msg.data, in[n:n+ln]))
	return n
}

func (msg *msgStreamTx) Size() int64 {
	return 8 + int64(len(msg.data))
}

func (msg *msgStreamTx) String() string {
	return "msgStreamTx"
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
