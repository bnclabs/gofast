package gofast

import "testing"

func TestCompression(t *testing.T) {
	flags := TransportFlag(EncodingProtobuf)
	flags = flags.SetRequest().SetStream().SetEndStream()

	flags = flags.SetSnappy()
	if flags.IsSnappy() == false {
		t.Fatalf("IsSnappy() failed")
	}
	flags = flags.SetGzip()
	if flags.IsGzip() == false {
		t.Fatalf("IsGzip() failed")
	}
	flags = flags.SetBzip2()
	if flags.IsBzip2() == false {
		t.Fatalf("IsBzip2() failed")
	}

	// make sure encoding has not changed.
	if flags.IsProtobuf() == false {
		t.Fatalf("Compression() disturbes encoding")
	}
	if flags.IsRequest() == false {
		t.Fatalf("Compression() disturbes request")
	}
	if flags.IsStream() == false {
		t.Fatalf("Compression() disturbes stream")
	}
	if flags.IsEndStream() == false {
		t.Fatalf("Compression() disturbes endstream")
	}
}

func TestEncoding(t *testing.T) {
	flags := TransportFlag(CompressionSnappy)
	flags = flags.SetRequest().SetStream().SetEndStream()

	flags = flags.SetProtobuf()
	if flags.IsProtobuf() == false {
		t.Fatalf("IsProtobuf() failed")
	}
	flags = flags.SetJson()
	if flags.IsJson() == false {
		t.Fatalf("IsJson() failed")
	}
	flags = flags.SetBson()
	if flags.IsBson() == false {
		t.Fatalf("IsBson() failed")
	}
	flags = flags.SetBinary()
	if flags.IsBinary() == false {
		t.Fatalf("IsBinary() failed")
	}

	// make sure encoding has not changed.
	if flags.IsSnappy() == false {
		t.Fatalf("Encoding() disturbes compression")
	}
	if flags.IsRequest() == false {
		t.Fatalf("Encoding() disturbes request")
	}
	if flags.IsStream() == false {
		t.Fatalf("Encoding() disturbes stream")
	}
	if flags.IsEndStream() == false {
		t.Fatalf("Encoding() disturbes endstream")
	}
}

func TestMisc(t *testing.T) {
	flags := TransportFlag(CompressionSnappy).SetProtobuf()

	flags = flags.SetRequest()
	if flags.IsRequest() == false {
		t.Fatalf("IsRequest() failed")
	}
	flags = flags.SetStream()
	if flags.IsStream() == false {
		t.Fatalf("IsStream() failed")
	}
	flags = flags.SetEndStream()
	if flags.IsEndStream() == false {
		t.Fatalf("IsEndStream() failed")
	}

	// make sure encoding has not changed.
	if flags.IsSnappy() == false {
		t.Fatalf("Misc() disturbes compression")
	}
	if flags.IsProtobuf() == false {
		t.Fatalf("Misc() disturbes encoding")
	}
}
