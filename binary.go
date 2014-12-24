package gofast

import "errors"

const (
	MtypeBinaryPayload uint16 = 0xF000
)

var ErrorInvalidMtype = errors.New("gofast.invalidMtype")

// BinaryEncoder implement default handlers for EncodingBinary.
type BinaryEncoder struct{}

// NewBinaryEncoder returns a new codec for EncodingBinary
func NewBinaryEncoder() *BinaryEncoder {
	return &BinaryEncoder{}
}

// Encode implements Encoder{} interface.
func (codec *BinaryEncoder) Encode(
	flags TransportFlag, opaque uint32,
	payload interface{}, out []byte) (data []byte, mtype uint16, err error) {

	s := payload.([]byte)
	copy(out, s)
	return out[:len(s)], MtypeBinaryPayload, nil
}

// Decode implements Encoder{} interface.
func (codec *BinaryEncoder) Decode(
	mtype uint16, flags TransportFlag, opaque uint32,
	data []byte) (payload interface{}, err error) {

	if mtype == MtypeBinaryPayload {
		out := make([]byte, len(data))
		copy(out, data)
		return out, nil
	}
	return nil, ErrorInvalidMtype
}
