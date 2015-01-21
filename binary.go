package gofast

// BinaryEncoder implement default handlers for EncodingBinary.
type BinaryEncoder struct{}

// NewBinaryEncoder returns a new codec for EncodingBinary
func NewBinaryEncoder() *BinaryEncoder {
	return &BinaryEncoder{}
}

// Encode implements Encoder{} interface.
func (codec *BinaryEncoder) Encode(
	flags TransportFlag, opaque uint32,
	payload interface{}, out []byte) (data []byte, err error) {

	s, ok := payload.([]byte)
	if !ok {
		return nil, ErrorBadPayload
	}
	copy(out, s)
	return out[:len(s)], nil
}

// Decode implements Encoder{} interface.
func (codec *BinaryEncoder) Decode(
	mtype uint16, flags TransportFlag, opaque uint32,
	data []byte) (payload interface{}, err error) {

	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
	return nil, ErrorInvalidMtype
}
