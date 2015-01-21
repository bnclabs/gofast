// flags:
//           +---------------+---------------+
//       byte|       0       |       1       |
//           +---------------+---------------+
//       bits|0 1 2 3 4 5 6 7|0 1 2 3 4 5 6 7|
//           +-------+-------+---------------+
//          0| COMP. |  ENC. |R S E          |
//           +-------+-------+---------------+
//
// COMP - compression algorithm to be applied on the payload
// ENC  - encoding to be applied on payload after compression
// R    - Request, if 1 a new request from client, else should always be 0
// S    - Streaming, if 1 - means receiver can expect more messages.
// E    - End-stream, if 1 - denotes end of stream, last message.
//
// * R, S, E are always interpreted in the context of opaque.
//
// Communicaton models, interpreting flags:
//
// +------------+------------------+--------+------------------+------------+
// |          Client               |  Flags |              Server           |
// +------------+------------------+--------+------------------+------------+
// |   Request  | Opaque-handling  | R S E | Opaque-handling  | Response   |
// +------------+------------------+-------+------------------+------------+
// |Post request>      ignore      > 1 0 0 >                  >            |
// |            |                  |       |      ignore      <Nothing     |
// |------------|------------------|-------|------------------|------------|
// |Single req. >      wclean      > 1 1 1 >      wclean      >            |
// |            <      forget      < 0 0 1 <      forget      <Single Resp.|
// |------------|------------------|-------|------------------|------------|
// |More msgs   >     remember     > 1 1 0 >     remember     >            |
// |            <         -        < 0 1 0 <        -         <More msgs   |
// |            <       forget     < 0 1 1 <      forget      <Last msg    |
// |------------|------------------|-------|------------------|------------|
// |More msgs   |     remember     | 1 1 0 |     remember     |            |
// |            |         -        | 0 1 0 |        -         |More msgs   |
// |            |       forget     | 0 1 1 |      forget      |Last msg    |
// |            |         -        | 0 1 0 |      ignore      |Ignore      |
// |            |       forget     | 0 0 1 |      ignore      |Ignore      |
// |------------|------------------|-------|------------------|------------|
// |More msgs   |     remember     | 1 1 0 |     remember     |            |
// |            |         -        | 0 1 0 |        -         |More msgs   |
// |            |      forget      | 0 0 1 |      forget      |            |
// |            |      ignore      | 0 1 0 |      ignore      |More msgs   |
// |            |      ignore      | 0 1 0 |      ignore      |More msgs   |
// +------------+------------------+-------+------------------+------------+
//
// Request notes,
// * {R:1} only client can send and only with a new or forgotten opaque value.
// * {R:1 S:0 E:1} is a post request no response expected from server.
// * {R:1 S:1 E:1} means req/response protocol.
// * {R:1 S:1 E:0} new request that can be closed by client.
// * {R:0 S:1 E:0} new streaming request from client.
// * {R:0 S:0 E:1} client closing the request.
// * {R:1 S:0 E:0} is not acceptable for request with a new opaque value.
//
// Response notes {R:0},
// * {S:1 E:0} means streaming response with more to come.
// * {S:x E:1} means server is closing the stream and request, may be with a
//   valid payload.

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

	// Stream sets packet for streaming
	Stream = 0x0200

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
	return flags.ClearStream() | TransportFlag(Stream)
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
	return (flags & TransportFlag(0x0200)) == Stream
}

// IsEndStream will set packet to denote end of current stream.
func (flags TransportFlag) IsEndStream() bool {
	return (flags & TransportFlag(0x0400)) == EndStream
}
