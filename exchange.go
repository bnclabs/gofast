package gofast

func unframe(frdata, out []byte, c *Config) {
	var start bool

	packeti = c.packetpool.Get()
	defer c.packetpool.Put(packeti)
	packet := packeti.([]byte)

	val, n := cbor2value(frdata, c)
	if _, ok := val.(CborPrefix); !ok {
		panic("expected cbor prefix")
	}
	// check for stream start
	if frdata[n] == 0x9f {
		start, n = true, n+1
	}

	// read opaque-tag
	byt := (frdata[n] & 0x1f) | cborType0 // fix as positive num
	opaque, m := cbor2valueM[byt](frdata[n:], c)
	n += m
	if opaque < tagOpaqueStart || opaque > tagOpaqueEnd {
		panic(fmt.Errorf("unexpected opaque value %v", opaque)
	}

	n = copy(packet, frdata[:n]) // frdata is not used anymore.

	datai = c.packetpool.Get()
	defer c.packetpool.Put(datai)
	data := datai.([]byte)

	var tag uint64
	for tag != tagMsg {
		tag, m = c.untag(packet[n:], data)
		n = copy(packet, data[:m])
	}
	c.unmessage(opaque, start, data[n:])
}

func unmessage(msg []byte, s *Stream, c *Config) (typ uint16, data []byte, n int) {
	if msg[n] != 0xbf {
		panic("expected a cbor map for message")
	}
	n++
	for msg[n] != 0xff {
		key, n1 := cbor2value(msg[n:], c)
		value, n2 := cbor2value(msg[n+n1:], c)
		switch key.(uint64) {
		case mhGzip:
			s.gzip(true)
		case mhLzw:
			s.gzip(true)
		case mhType:
			typ = value
		case mhMsg:
			data = value
		}
		n += n1 + n2
	}
	return typ, data
}
