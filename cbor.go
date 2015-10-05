//  Copyright (c) 2015 Couchbase, Inc.

package gofast

// cborUndefined type as part of simple-type codepoint-23.
type cborUndefined byte

// cborIndefinite code, first-byte of stream encoded data items.
type cborIndefinite byte

// cborBreakStop code, last-byte of stream encoded the data items.
type cborBreakStop byte

// cborPrefix tagged-type, a byte-string of cbor data-item that
// will be wrapped with a unique prefix before sending out.
type cborPrefix []byte

// cbor tagged-type, a byte-string of cbor data-item.
type cborCbor []byte

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

func cborMajor(b byte) byte {
	return b & 0xe0
}

func cborInfo(b byte) byte {
	return b & 0x1f
}

func cborHdr(major, info byte) byte {
	return (major & 0xe0) | (info & 0x1f)
}

const ( // pre-defined tag values
	tagDateTime        = iota // datetime as utf-8 string
	tagEpoch                  // datetime as +/- int or +/- float
	tagPosBignum              // as []bytes
	tagNegBignum              // as []bytes
	tagDecimalFraction        // decimal fraction as array of [2]num
	tagBigFloat               // as array of [2]num
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
	// tag 40 (unassigned as per spec). place-holder for "id" header key.
	tagData
	// tag 41 (unassigned as per spec). says payload is compressed using
	// Gzip compression method.
	tagGzip
	// tag 42 (unassinged as per spec). says payload is compressed using
	// Lzw compression method.
	tagLzw

	// unassigned 38..255

	// opaque-space 256..55798
	tagOpaqueStart = iota + 235
	tagOpaqueEnd   = iota + 55776

	tagCborPrefix

	// unassigned 55800..
)

var brkstp byte = cborHdr(cborType7, cborItemBreak)

var hdrIndefiniteBytes = cborHdr(cborType2, cborIndefiniteLength)
var hdrIndefiniteText = cborHdr(cborType3, cborIndefiniteLength)
var hdrIndefiniteArray = cborHdr(cborType4, cborIndefiniteLength)
var hdrIndefiniteMap = cborHdr(cborType5, cborIndefiniteLength)
