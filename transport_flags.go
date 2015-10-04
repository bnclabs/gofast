//  Copyright (c) 2014 Couchbase, Inc.

package gofast

// TransportFlag tell packet encoding and compression formats.
type TransportFlag uint16

//------------------
// Compression flags
//------------------

const (
	// CompressionNone does not apply compression on the payload.
	CompressionNone TransportFlag = 0x0000

	// CompressionSnappy apply snappy compression on payload.
	CompressionSnappy = 0x0001

	// CompressionGzip apply gzip compression on payload.
	CompressionGzip = 0x0002

	// CompressionBzip2 apply bzip2 compression on the payload.
	CompressionBzip2 = 0x0003

	// CompressionLZW apply LZW compression on the payload.
	CompressionLZW = 0x0004
)

// GetCompression returns the compression bits from flags.
func (flags TransportFlag) GetCompression() TransportFlag {
	return flags & TransportFlag(0x000F)
}

// SetCompression will set packet compression.
func (flags TransportFlag) SetCompression(flag TransportFlag) TransportFlag {
	return (flags & TransportFlag(0xFFF0)) | flag
}

// IsSnappy returns true if compression on the payload is snappy.
func (flags TransportFlag) IsSnappy() bool {
	return byte(flags&TransportFlag(0x000F)) == CompressionSnappy
}

// IsGzip() returns true if compression on payload is gzip.
func (flags TransportFlag) IsGzip() bool {
	return byte(flags&TransportFlag(0x000F)) == CompressionGzip
}

// IsBzip2 returns true if compression on payload is Bzip2.
func (flags TransportFlag) IsBzip2() bool {
	return byte(flags&TransportFlag(0x000F)) == CompressionBzip2
}

// IsLZW returns true if compression on payload is LZW.
func (flags TransportFlag) IsLZW() bool {
	return byte(flags&TransportFlag(0x000F)) == CompressionLZW
}

// SetSnappy will set packet compression to snappy
func (flags TransportFlag) SetSnappy() TransportFlag {
	return (flags & TransportFlag(0xFFF0)) | TransportFlag(CompressionSnappy)
}

// SetGzip will set packet compression to Gzip
func (flags TransportFlag) SetGzip() TransportFlag {
	return (flags & TransportFlag(0xFFF0)) | TransportFlag(CompressionGzip)
}

// SetBzip2 will set packet compression to bzip2
func (flags TransportFlag) SetBzip2() TransportFlag {
	return (flags & TransportFlag(0xFFF0)) | TransportFlag(CompressionBzip2)
}

// SetLZW will set packet compression to LZW
func (flags TransportFlag) SetLZW() TransportFlag {
	return (flags & TransportFlag(0xFFF0)) | TransportFlag(CompressionLZW)
}

//---------------
// Encoding flags
//---------------

const (
	// EncodingNone does not apply any encoding/decoding on the payload.
	EncodingNone TransportFlag = 0x0000

	// EncodingProtobuf uses protobuf as coding format.
	EncodingProtobuf = 0x0010

	// EncodingJson uses JSON as coding format.
	EncodingJson = 0x0020

	// EncodingBson uses BSON as coding format.
	EncodingBson = 0x0030

	// EncodingBinary uses binary as coding format.
	EncodingBinary = 0x0040
)

// GetEncoding will get the encoding bits from flags
func (flags TransportFlag) GetEncoding() TransportFlag {
	return flags & TransportFlag(0x00F0)
}

// SetEncoding will set packet encoding.
func (flags TransportFlag) SetEncoding(flag TransportFlag) TransportFlag {
	return (flags & TransportFlag(0xFF0F)) | flag
}

// IsProtobuf return true if payload encoding is using protobuf.
func (flags TransportFlag) IsProtobuf() bool {
	return (flags & TransportFlag(0x00F0)) == EncodingProtobuf
}

// IsJson return true if payload encoding is using JSON.
func (flags TransportFlag) IsJson() bool {
	return (flags & TransportFlag(0x00F0)) == EncodingJson
}

// IsBson return true if payload encoding is using BSON.
func (flags TransportFlag) IsBson() bool {
	return (flags & TransportFlag(0x00F0)) == EncodingBson
}

// IsBinary return true if payload encoding is using raw-binary.
func (flags TransportFlag) IsBinary() bool {
	return (flags & TransportFlag(0x00F0)) == EncodingBinary
}

// SetProtobuf will set packet encoding to protobuf
func (flags TransportFlag) SetProtobuf() TransportFlag {
	return (flags & TransportFlag(0xFF0F)) | TransportFlag(EncodingProtobuf)
}

// SetJson will set packet encoding to protobuf
func (flags TransportFlag) SetJson() TransportFlag {
	return (flags & TransportFlag(0xFF0F)) | TransportFlag(EncodingJson)
}

// SetBson will set packet encoding to protobuf
func (flags TransportFlag) SetBson() TransportFlag {
	return (flags & TransportFlag(0xFF0F)) | TransportFlag(EncodingBson)
}

// SetBinary will set packet encoding to protobuf
func (flags TransportFlag) SetBinary() TransportFlag {
	return (flags & TransportFlag(0xFF0F)) | TransportFlag(EncodingBinary)
}

//------------
// Misc. Flags
//------------

const (
	// Request packet will set direction as request, client->server.
	Request TransportFlag = 0x0100

	// TStream sets packet for streaming
	TStream = 0x0200

	// EndStream denotes last packet in a stream.
	EndStream = 0x0400
)

// ClearRequest will set packet as request.
func (flags TransportFlag) ClearRequest() TransportFlag {
	return flags & TransportFlag(0xFEFF)
}

// SetRequest will set packet as request.
func (flags TransportFlag) SetRequest() TransportFlag {
	return flags.ClearRequest() | TransportFlag(Request)
}

// ClearStream will set packet for streaming.
func (flags TransportFlag) ClearStream() TransportFlag {
	return flags & TransportFlag(0xFDFF)
}

// SetStream will set packet for streaming.
func (flags TransportFlag) SetStream() TransportFlag {
	return flags.ClearStream() | TransportFlag(TStream)
}

// ClearEndStream will set packet to denote end of current stream.
func (flags TransportFlag) ClearEndStream() TransportFlag {
	return flags & TransportFlag(0xFBFF)
}

// SetEndStream will set packet to denote end of current stream.
func (flags TransportFlag) SetEndStream() TransportFlag {
	return flags.ClearEndStream() | TransportFlag(EndStream)
}

// IsRequest will set packet as request.
func (flags TransportFlag) IsRequest() bool {
	return (flags & TransportFlag(0x0100)) == Request
}

// IsStream will set packet for streaming.
func (flags TransportFlag) IsStream() bool {
	return (flags & TransportFlag(0x0200)) == TStream
}

// IsEndStream will set packet to denote end of current stream.
func (flags TransportFlag) IsEndStream() bool {
	return (flags & TransportFlag(0x0400)) == EndStream
}
