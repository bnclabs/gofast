package gofast

import "fmt"

// Version64 example version implementation.
type Version64 uint64

// Less implement Version interface{}.
func (v *Version64) Less(other Version) bool {
	return (*v) < *(other.(*Version64))
}

// Equal implement Version interface{}.
func (v *Version64) Equal(other Version) bool {
	return (*v) == *(other.(*Version64))
}

// Encode implement Version interface{}.
func (v *Version64) Encode(out []byte) []byte {
	out = fixbuffer(out, v.Size())
	n := 0
	n += valuint642cbor(uint64(*v), out[n:])
	return out[:n]
}

// Decode implement Version interface{}.
func (v *Version64) Decode(in []byte) int64 {
	val, n := cborItemLength(in)
	*v = Version64(val)
	return int64(n)
}

// Size implement Version interface{}.
func (v *Version64) Size() int64 {
	return 9
}

// String implement Version interface{}.
func (v *Version64) String() string {
	return fmt.Sprintf("Version64(%v)", *v)
}
