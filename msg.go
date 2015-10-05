package gofast

type Message interface {
	Id() uint64
	Encode(out []byte) int
	Decode(in []byte)
	Free()
}

const (
	MsgPing             = 0xFFFFFFFFFFFFFFFF
	MsgWhoami    uint64 = 0xFFFFFFFFFFFFFFFE
	MsgHeartbeat        = 0x1
)

type Version interface {
	Less() bool
	Equal() bool
	String() string
	Value() interface{}
}
