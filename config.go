package gofast

import "compress/flate"
import s "github.com/prataprc/gosettings"

// DefaultSettings for gofast, start and end arguments are used to generate
// opaque id for streams.
//
// Configurable parameters:
//
// "buffersize" (int64, default: 512)
//      maximum size that a single message will need for encoding.
//
// "batchsize" (int64, default:1 )
//		number of messages to batch before writing to socket, transport
//		will create a local buffer of size buffersize * batchsize.
//
// "chansize" (int64, default: 100000)
//		buffered channel size to use for internal go-routines.
//
// "opaque.start" (int64, default: <start-argument>)
//		starting opaque range, inclusive. must be > 255
//
// "opaque.end" (int64, default: <end-argument>)
//		ending opaque range, inclusive.
//
// "tags" (int64, default: "")
//    comma separated list of tags to apply, in specified order.
//
// "gzip.level" (int64, default: <flate.BestSpeed>)
//    gzip compression level, if `tags` contain "gzip".
//
func DefaultSettings(start, end int64) s.Settings {
	return s.Settings{
		"buffersize":   512,
		"batchsize":    1,
		"chansize":     100000,
		"tags":         "",
		"opaque.start": start,
		"opaque.end":   end,
		"gzip.level":   flate.BestSpeed,
	}
}
