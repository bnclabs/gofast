package gofast

import _ "fmt"

// BinaryEncoder implement default handlers for EncodingBinary.
type BinaryEncoder struct{}

// NewBinaryEncoder returns a new codec for EncodingBinary
func NewBinaryEncoder() *BinaryEncoder {
	return &BinaryEncoder{}
}

// Encode implements Encoder{} interface.
func (codec *BinaryEncoder) Encode(
	flags TransportFlag, opaque uint32,
	payload interface{}, out []byte) ([]byte, error) {

	if payload == nil {
		return []byte{}, nil
	}

	var data []byte

	switch bs := payload.(type) {
	case string:
		data = []byte(bs)
	case []byte:
		data = bs
	default:
		return nil, ErrorBadPayload
	}
	copy(out, data)
	return out[:len(data)], nil
}

// Decode implements Encoder{} interface.
func (codec *BinaryEncoder) Decode(
	mtype uint16, flags TransportFlag, opaque uint32,
	data []byte) (payload interface{}, err error) {

	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}
