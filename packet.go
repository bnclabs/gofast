package gofast

func message(msg Messager, out []byte) int {
	n := tag2cbor(tagMsg, out)
	n += msg.Encode(out[n:])
	return n
}

func unmessage(payload, out []byte) int {
}

func gzip(payload, out []byte) int {
}

func ungzip(payload, out []byte) int {
}

func lzw(payload, out []byte) int {
}

func unlzw(payload, out []byte) int {
}
