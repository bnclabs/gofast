//  Copyright (c) 2015 Couchbase, Inc.

package gofast

import "fmt"

// CborContainerEncoding, encoding method to use for arrays and maps.
type CborContainerEncoding byte

const (
	// LengthPrefix encoding for composite types. That is, for arrays and maps
	// encode the number of contained items as well.
	LengthPrefix CborContainerEncoding = iota + 1

	// Stream encoding for composite types. That is, for arrays and maps
	// use cbor's indefinite and break-stop to encode member items.
	Stream
)

// Config and access gson functions. All APIs to gson is defined via
// config. To quickly get started, use NewDefaultConfig() that will
// create a configuration with default values.
type Config struct {
	ct CborContainerEncoding
}

// NewDefaultConfig returns a new configuration with default values.
// CborContainerEncoding: Stream
func NewDefaultConfig() *Config {
	return &Config{ct: Stream}
}

// ContainerEncoding for cbor.
func (config Config) ContainerEncoding(ct CborContainerEncoding) *Config {
	config.ct = ct
	return &config
}

// SmallintToCbor encode tiny integers between -23..+23.
// Can be used by libraries that build on top of cbor.
func (config *Config) SmallintToCbor(item int8, out []byte) int {
	if item < 0 {
		out[0] = cborHdr(cborType1, byte(-(item + 1))) // -23 to -1
	} else {
		out[0] = cborHdr(cborType0, byte(item)) // 0 to 23
	}
	return 1
}

// SimpletypeToCbor that falls outside golang native type,
// code points 0..19 and 32..255 are un-assigned.
// Can be used by libraries that build on top of cbor.
func (config *Config) SimpletypeToCbor(typcode byte, out []byte) int {
	return simpletypeToCbor(typcode, out)
}

// IsIndefiniteBytes can be used to check the shape of cbor
// data-item, like byte-string, string, array or map, that
// is going to come afterwards.
// Can be used by libraries that build on top of cbor.
func (config *Config) IsIndefiniteBytes(b CborIndefinite) bool {
	return b == CborIndefinite(hdrIndefiniteBytes)
}

// IsIndefiniteText can be used to check the shape of cbor
// data-item, like byte-string, string, array or map, that
// is going to come afterwards.
// Can be used by libraries that build on top of cbor.
func (config *Config) IsIndefiniteText(b CborIndefinite) bool {
	return b == CborIndefinite(hdrIndefiniteText)
}

// IsIndefiniteArray can be used to check the shape of cbor
// data-item, like byte-string, string, array or map, that
// is going to come afterwards.
// Can be used by libraries that build on top of cbor.
func (config *Config) IsIndefiniteArray(b CborIndefinite) bool {
	return b == CborIndefinite(hdrIndefiniteArray)
}

// IsIndefiniteMap can be used to check the shape of cbor
// data-item, like byte-string, string, array or map, that
// is going to come afterwards.
// Can be used by libraries that build on top of cbor.
func (config *Config) IsIndefiniteMap(b CborIndefinite) bool {
	return b == CborIndefinite(hdrIndefiniteMap)
}

// IsBreakstop can be used to check whether chunks of
// cbor bytes, or texts, or array items or map items
// ending with the current byte. Can be used by libraries
// that build on top of cbor.
func (config *Config) IsBreakstop(b byte) bool {
	return b == brkstp
}

// ValueToCbor golang data into cbor binary.
func (config *Config) ValueToCbor(item interface{}, out []byte) int {
	return value2cbor(item, out, config)
}

// MapsliceToCbor to encode key,value pairs into cbor
func (config *Config) MapsliceToCbor(items [][2]interface{}, out []byte) int {
	return mapl2cbor(items, out, config)
}

// CborToValue cbor binary into golang data.
func (config *Config) CborToValue(buf []byte) (interface{}, int) {
	return cbor2value(buf, config)
}

func (config *Config) ConfigString() string {
	return fmt.Sprintf("ct:%v", config.ct)
}

func (ct CborContainerEncoding) String() string {
	switch ct {
	case LengthPrefix:
		return "LengthPrefix"
	case Stream:
		return "Stream"
	default:
		panic("new space-kind")
	}
}
