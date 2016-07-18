package gofast

import "compress/flate"

func DefaultSettings(start, end uint64) map[string]interface{} {
	return map[string]interface{}{
		"buffersize":   uint64(512),
		"chansize":     uint64(100000),
		"batchsize":    uint64(1),
		"tags":         "",
		"opaque.start": start,
		"opaque.end":   end,
		"gzip.level":   flate.BestSpeed,
	}
}
