package gofast

import "compress/flate"

func DefaultSettings(start, end int64) Settings {
	return Settings{
		"buffersize":   512,
		"chansize":     100000,
		"batchsize":    1,
		"tags":         "",
		"opaque.start": start,
		"opaque.end":   end,
		"gzip.level":   flate.BestSpeed,
	}
}
