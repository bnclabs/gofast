package main

import "encoding/binary"

const cborMaxSmallInt = 23

const ( // major types.
	cborType0 byte = iota << 5 // unsigned integer
	cborType1                  // negative integer
	cborType2                  // byte string
	cborType3                  // text string
	cborType4                  // array
	cborType5                  // map
	cborType6                  // tagged data-item
	cborType7                  // floating-point, simple-types and break-stop
)

const ( // associated information for type0 and type1.
	// 0..23 actual value
	cborInfo24 byte = iota + 24 // followed by 1-byte data-item
	cborInfo25                  // followed by 2-byte data-item
	cborInfo26                  // followed by 4-byte data-item
	cborInfo27                  // followed by 8-byte data-item
	// 28..30 reserved
	cborIndefiniteLength = 31 // for byte-string, string, arr, map
)

const ( // simple types for type7
	// 0..19 unassigned
	cborSimpleTypeFalse byte = iota + 20 // encodes nil type
	cborSimpleTypeTrue
	cborSimpleTypeNil
	cborSimpleUndefined
	cborSimpleTypeByte // the actual type in next byte 32..255
	cborFlt16          // IEEE 754 Half-Precision Float
	cborFlt32          // IEEE 754 Single-Precision Float
	cborFlt64          // IEEE 754 Double-Precision Float
	// 28..30 reserved
	cborItemBreak = 31 // stop-code for indefinite-length items
)

const ( // pre-defined tag values
	tagDateTime        uint64 = iota // datetime as utf-8 string
	tagEpoch                         // datetime as +/- int or +/- float
	tagPosBignum                     // as []bytes
	tagNegBignum                     // as []bytes
	tagDecimalFraction               // decimal fraction as array of [2]num
	tagBigFloat                      // as array of [2]num

	// unassigned 6..20

	// TODO: tagBase64URL, tagBase64, tagBase16
	tagBase64URL = iota + 15 // interpret []byte as base64 format
	tagBase64                // interpret []byte as base64 format
	tagBase16                // interpret []byte as base16 format
	tagCborEnc               // embedd another CBOR message

	// unassigned 25..31

	tagURI          = iota + 22 // defined in rfc3986
	tagBase64URLEnc             // base64 encoded url as text strings
	tagBase64Enc                // base64 encoded byte-string as text strings
	tagRegexp                   // PCRE and ECMA262 regular expression
	tagMime                     // MIME defined by rfc2045

	// tag 37 (unassigned as per spec). says payload is encoded message
	// that shall be passed on to the subscribed handler.
	tagMsg
	// tag 38 (unassigned as per spec). place-holder for "version" header key.
	tagVersion
	// tag 39 (unassigned as per spec). place-holder for "id" header key.
	tagId
	// tag 40 (unassigned as per spec). place-holder for "data" header key.
	tagData
	// tag 41 (unassigned as per spec). says payload is compressed using
	// Gzip compression method.
	tagGzip
	// tag 42 (unassinged as per spec). says payload is compressed using
	// Lzw compression method.
	tagLzw

	// opaque-space 256..55798
	tagOpaqueStart = iota + 235
	tagOpaqueEnd   = iota + 55776

	tagCborPrefix

	// unassigned 55800..
)

var brkstp byte = cborHdr(cborType7, cborItemBreak)

func cborMajor(b byte) byte {
	return b & 0xe0
}

func cborInfo(b byte) byte {
	return b & 0x1f
}

func cborHdr(major, info byte) byte {
	return (major & 0xe0) | (info & 0x1f)
}

func tag2cbor(tag uint64, buf []byte) int {
	n := valuint642cbor(tag, buf)
	buf[0] = (buf[0] & 0x1f) | cborType6 // fix the type as tag.
	return n
}

func valuint82cbor(item byte, buf []byte) int {
	if item <= cborMaxSmallInt {
		buf[0] = cborHdr(cborType0, item) // 0..23
		return 1
	}
	buf[0] = cborHdr(cborType0, cborInfo24)
	buf[1] = item // 24..255
	return 2
}

func valuint162cbor(item uint16, buf []byte) int {
	if item < 256 {
		return valuint82cbor(byte(item), buf)
	}
	buf[0] = cborHdr(cborType0, cborInfo25)
	binary.BigEndian.PutUint16(buf[1:], item) // 256..65535
	return 3
}

func valuint322cbor(item uint32, buf []byte) int {
	if item < 65536 {
		return valuint162cbor(uint16(item), buf) // 0..65535
	}
	buf[0] = cborHdr(cborType0, cborInfo26)
	binary.BigEndian.PutUint32(buf[1:], item) // 65536 to 4294967295
	return 5
}

func valuint642cbor(item uint64, buf []byte) int {
	if item < 4294967296 {
		return valuint322cbor(uint32(item), buf) // 0..4294967295
	}
	// 4294967296 .. 18446744073709551615
	buf[0] = cborHdr(cborType0, cborInfo27)
	binary.BigEndian.PutUint64(buf[1:], item)
	return 9
}

func valbytes2cbor(item []byte, buf []byte) int {
	n := valuint642cbor(uint64(len(item)), buf)
	buf[0] = (buf[0] & 0x1f) | cborType2 // fix the type from type0->type2
	copy(buf[n:], item)
	return n + len(item)
}

func valtext2cbor(item string, buf []byte) int {
	n := valbytes2cbor([]byte(item), buf)
	buf[0] = (buf[0] & 0x1f) | cborType3 // fix the type from type2->type3
	return n
}

func arrayStart(buf []byte) int {
	// indefinite length array
	buf[0] = cborHdr(cborType4, byte(cborIndefiniteLength))
	return 1
}

func mapStart(buf []byte) int {
	// indefinite length map
	buf[0] = cborHdr(cborType5, byte(cborIndefiniteLength))
	return 1
}

func breakStop(buf []byte) int {
	// break stop for indefinite array or map
	buf[0] = cborHdr(cborType7, byte(cborItemBreak))
	return 1
}

func cborItemLength(buf []byte) (int, int) {
	if y := cborInfo(buf[0]); y < cborInfo24 {
		return int(y), 1
	} else if y == cborInfo24 {
		return int(buf[1]), 2
	} else if y == cborInfo25 {
		return int(binary.BigEndian.Uint16(buf[1:])), 3
	} else if y == cborInfo26 {
		return int(binary.BigEndian.Uint32(buf[1:])), 5
	}
	return int(binary.BigEndian.Uint64(buf[1:])), 9 // info27
}
