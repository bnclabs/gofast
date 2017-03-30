package gofast

import "fmt"
import "sync/atomic"

// | 0xd9 0xd9f7 | 0xc6 | packet |
func (t *Transport) post(msg Message, stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txpost, 1)
	n = tag2cbor(tagCborPrefix, out)      // prefix
	out[n] = 0xc6                         // 0xc6 (post, 0b100_00110 <tag,6>
	n++                                   //
	n += t.framepkt(msg, stream, out[n:]) // packet
	return n
}

// | 0xd9 0xd9f7 | 0x81 | packet |
func (t *Transport) request(msg Message, stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txreq, 1)
	n = tag2cbor(tagCborPrefix, out)      // prefix
	out[n] = 0x81                         // 0x81 (request, 0b100_10001 <arr,1>)
	n += 1                                //
	n += t.framepkt(msg, stream, out[n:]) // packet
	return n
}

// | 0xd9 0xd9f7 | 0x81 | packet |
func (t *Transport) response(msg Message, stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txresp, 1)
	n = tag2cbor(tagCborPrefix, out)      // prefix
	out[n] = 0x81                         // 0x81 (response, 0b100_10001 <arr,1>)
	n += 1                                //
	n += t.framepkt(msg, stream, out[n:]) // packet
	return n
}

// | 0xd9 0xd9f7  | 0x9f | packet2    |
func (t *Transport) start(msg Message, stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txstart, 1)
	n = tag2cbor(tagCborPrefix, out)      // prefix
	n += arrayStart(out[n:])              // 0x9f (start stream as cbor array)
	n += t.framepkt(msg, stream, out[n:]) // packet
	return n
}

// | 0xd9 0xd9f7  | 0xc7 | packet2    |
func (t *Transport) stream(msg Message, stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txstream, 1)
	n = tag2cbor(tagCborPrefix, out)      // prefix
	out[n] = 0xc7                         // 0xc7 (stream msg, 0b110_00111 <tag,7>)
	n += 1                                //
	n += t.framepkt(msg, stream, out[n:]) // packet
	return n
}

// | 0xd9 0xd9f7  | 0xc8 | end-packet |
func (t *Transport) finish(stream *Stream, out []byte) (n int) {
	atomic.AddUint64(&t.n_txfin, 1)
	var scratch [16]byte
	n = tag2cbor(tagCborPrefix, out)         // prefix
	out[n] = 0xc8                            // 0xc7 (end stream, 0b110_01000 <tag,8>)
	n += 1                                   //
	m := tag2cbor(stream.opaque, scratch[:]) // tag-opaque
	scratch[m] = 0xff                        // 0xff (payload)
	m += 1
	n += valbytes2cbor(scratch[:m], out[n:]) // packet
	return n
}

func (t *Transport) framepkt(msg Message, stream *Stream, ping []byte) (n int) {
	data, pong := stream.data, stream.tagout

	// tagMsg
	n = tag2cbor(tagMsg, ping) // tagMsg
	n += mapStart(ping[n:])
	n += valuint642cbor(tagId, ping[n:])    // hdr-tagId
	n += valuint642cbor(msg.ID(), ping[n:]) // value
	n += valuint642cbor(tagData, ping[n:])  // hdr-tagData
	data = msg.Encode(data)                 // value
	n += valbytes2cbor(data, ping[n:])
	n += breakStop(ping[n:])

	// NOTE: tagenc is updated as part of whoamiMsg message, due to
	// which it needs to be skipped for whoamiMsg message during
	// handshake, for now we skip tagenc for whoamiMsg all the time.
	if _, ok := msg.(*whoamiMsg); !ok {
		for tag, fn := range t.tagenc { // roll up tags
			m := fn(ping[:n], pong)
			if m == 0 { // skip tag
				continue
			}
			n = tag2cbor(tag, ping)
			n += valbytes2cbor(pong[:m], ping[n:])
		}
	}

	m := tag2cbor(stream.opaque, pong) // finally roll up opaque
	m += valbytes2cbor(ping[:n], pong[m:])
	n = valbytes2cbor(pong[:m], ping) // packet encoded as CBOR byte array
	return n
}

type txproto struct {
	packet []byte // request
	flush  bool
	async  bool
	n      int // response
	err    error
	respch chan *txproto
}

func (t *Transport) tx(out []byte, flush bool) (err error) {
	arg := t.fromtxpool()
	defer func() {
		arg.packet = arg.packet[:cap(arg.packet)]
		t.p_txcmd <- arg
	}()

	n := copy(arg.packet, out)
	arg.packet, arg.flush, arg.async = arg.packet[:n], flush, false
	arg.respch = make(chan *txproto, 1)
	select {
	case t.txch <- arg:
		select {
		case resp := <-arg.respch:
			n, err := resp.n, resp.err
			if err == nil && n != len(arg.packet) {
				return fmt.Errorf("partial write")
			}
			return err // success or error
		case <-t.killch:
			return fmt.Errorf("transport closed")
		}

	case <-t.killch:
		return fmt.Errorf("transport closed")
	}
	return nil
}

func (t *Transport) txasync(out []byte, flush bool) (err error) {
	arg := t.fromtxpool()
	n := copy(arg.packet, out)
	arg.packet = arg.packet[:n]

	arg.flush, arg.async = flush, true
	select {
	case t.txch <- arg:
	case <-t.killch:
		return fmt.Errorf("transport closed")
	}
	return nil
}
