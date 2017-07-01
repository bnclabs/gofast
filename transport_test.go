package gofast

import "testing"
import "reflect"
import "fmt"
import "syscall"
import "sort"
import "net"
import "time"
import "sync"
import "strings"

import s "github.com/prataprc/gosettings"
import "github.com/prataprc/golog"

func TestTransport(t *testing.T) {
	ver, addr := testVersion(1), <-testBindAddrs
	lis, serverch := newServer("server", addr, "gzip")      // init server
	transc := newClient("client", addr, "gzip").Handshake() // init client
	transv := <-serverch

	cCounts := transv.Stat()
	sCounts := transv.Stat()
	// test
	if ref := "server"; transv.Name() != ref {
		t.Errorf("expected %v, got %v", ref, transv.Name())
	} else if tags, ok := transc.tagenc[tagGzip]; !ok && len(transc.tagenc) != 1 {
		t.Errorf("expected gzip, got %v, %v", tags, len(transc.tagenc))
	} else if ref := "client"; ref != transc.Name() {
		t.Errorf("expected %v, got %v", ref, transc.Name())
	} else if !transc.PeerVersion().Equal(&ver) {
		t.Errorf("expected %v, got %v", ver, transc.peerver.Load().(Version))
	} else if s := transc.RemoteAddr().String(); s != addr {
		t.Errorf("expected %v, got %v", s, addr)
	} else if !verify(cCounts, "n_flushes", "n_rx", "n_tx", 2) {
		t.Errorf("unexpected cCounts: %v", cCounts)
	} else if !verify(cCounts, "n_rxreq", "n_rxresp", 1) {
		t.Errorf("unexpected cCounts: %v", cCounts)
	} else if !verify(cCounts, "n_txresp", 1, "n_rxbyte", "n_txbyte", 90) {
		t.Errorf("unexpected cCounts: %v", cCounts)
	} else if !verify(sCounts, "n_flushes", "n_rx", "n_tx", 2) {
		t.Errorf("unexpected sCounts: %v", sCounts)
	} else if !verify(sCounts, "n_rxreq", "n_rxresp", 1) {
		t.Errorf("unexpected cCounts: %v", cCounts)
	} else if !verify(sCounts, "n_txresp", 1, "n_rxbyte", "n_txbyte", 90) {
		t.Errorf("unexpected sCounts: %v", sCounts)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestSubscribeMessage(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	// test
	if ref := "server"; transv.Name() != ref {
		t.Errorf("expected %v, got %v", ref, transv.Name())
	}
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic")
			}
		}()
		transc.SubscribeMessage(newPing("should failed"), nil)
	}()

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestFlushPeriod(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	// test
	transc.FlushPeriod(10 * time.Millisecond) // 99 flushes + 1 from handshake
	time.Sleep(2 * time.Second)
	cCounts := transc.Stat()
	if ref, n := uint64(10), cCounts["n_flushes"]; n < ref {
		t.Errorf("expected less than %v, got %v", ref, n)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestHeartbeat(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	// test
	transc.SendHeartbeat(10 * time.Millisecond)
	time.Sleep(1 * time.Second)
	transc.Close()
	time.Sleep(1 * time.Second)
	cCounts := transc.Stat()
	sCounts := transv.Stat()

	if !verify(cCounts, "n_txreq", "n_rxresp", 1, "n_rx", 2) {
		t.Errorf("unexpected cCounts %v", cCounts)
	} else if limit := uint64(5); cCounts["n_flushes"] < limit {
		t.Errorf("atleast %v, got %v", limit, cCounts["n_flushes"])
	} else if !verify(cCounts, "n_tx", "n_flushes") {
		t.Errorf("failed cCounts: %v", cCounts)
	}
	if !verify(sCounts, "n_rxreq", "n_txresp", 1, "n_flushes", "n_tx", 2) {
		t.Errorf("unexpected sCounts %v", sCounts)
	} else if cCounts["n_rxbyte"] != sCounts["n_txbyte"] {
		t.Errorf("mismatch %v, %v", cCounts["n_rxbyte"], sCounts["n_txbyte"])
	} else if cCounts["n_txbyte"] != sCounts["n_rxbyte"] {
		t.Errorf("mismatch %v, %v", cCounts["n_txbyte"], sCounts["n_rxbyte"])
	}
	if x, y := cCounts["n_flushes"], sCounts["n_rx"]; x != y && x != (y+1) {
		t.Errorf("mismatch %v, %v", x, y)
	} else if !verify(sCounts, "n_rxbeats", "n_rxpost") {
		t.Errorf("mismatch %v, %v", sCounts["n_rxbeats"], sCounts["n_rxpost"])
	} else if n := sCounts["n_rxpost"]; n != 99 && n != 100 && n != 98 {
		t.Errorf("not 100, not 99, not 98: %v", n)
	}

	ref, since := (1100 * time.Millisecond), transv.Silentsince()
	if since > ref {
		t.Errorf("expected less than %v, got %v", ref, since)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transv.Close()
}

func TestPing(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch

	// test
	refs := "hello world"
	if echo, err := transc.Ping(refs); err != nil {
		t.Error(err)
	} else if echo != refs {
		t.Errorf("expected atleast %v, got %v", refs, echo)
	}
	counts := transc.Stat()
	if ref, n := uint64(3), counts["n_flushes"]; n != ref {
		t.Errorf("expected atleast %v, got %v", ref, n)
	} else if n = counts["n_rx"]; n != ref {
		t.Errorf("expected atleast %v, got %v", ref, n)
	} else if n = counts["n_tx"]; n != ref {
		t.Errorf("expected atleast %v, got %v", ref, n)
	} else if ref, n = 2, counts["n_txreq"]; n != ref {
		t.Errorf("expected atleast %v, got %v", ref, n)
	} else if n = counts["n_rxresp"]; n != ref {
		t.Errorf("expected atleast %v, got %v", ref, n)
	} else if x, y := counts["n_rxbyte"], counts["n_txbyte"]; x != y {
		t.Errorf("expected atleast %v, got %v", x, y)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestWhoami(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	wai, err := transc.Whoami()
	if err != nil {
		t.Error(err)
	} else if string(wai.name) != "server" {
		t.Errorf("expected %v, got %v", "server", string(wai.name))
	}
	counts := transc.Stat()
	if ref, n := uint64(3), counts["n_flushes"]; n != ref {
		t.Errorf("expected atleast %v, got %v", ref, n)
	} else if n = counts["n_tx"]; n != ref {
		t.Errorf("expected atleast %v, got %v", ref, n)
	} else if n = counts["n_rx"]; n != ref {
		t.Errorf("expected atleast %v, got %v", ref, n)
	} else if ref, n = 2, counts["n_txreq"]; n != ref {
		t.Errorf("expected atleast %v, got %v", ref, n)
	} else if n = counts["n_rxresp"]; n != ref {
		t.Errorf("expected atleast %v, got %v", ref, n)
	} else if x, y := counts["n_rxbyte"], counts["n_txbyte"]; x != y {
		t.Errorf("expected atleast %v, got %v", x, y)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestTransPost(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	msg := &testMessage{1234}
	donech := make(chan bool, 2)
	transc.SubscribeMessage(
		&testMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m testMessage

			m.Decode(rxmsg.Data)
			if s != nil {
				t.Errorf("expected nil, got %v", s)
			} else if !reflect.DeepEqual(&m, msg) {
				t.Errorf("expected %v, got %v", msg, m)
			}
			donech <- true
			return nil
		})
	transv.SubscribeMessage(
		&testMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m testMessage

			m.Decode(rxmsg.Data)
			if s != nil {
				t.Errorf("expected nil, got %v", s)
			} else if !reflect.DeepEqual(&m, msg) {
				t.Errorf("expected %#v, got %#v", msg, m)
			}
			transv.Post(&m, true)
			return nil
		})
	transc.Post(msg, true)
	<-donech

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestTransPostEmpty(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	msg := &emptyMessage{}
	donech := make(chan bool, 2)
	transc.SubscribeMessage(
		&emptyMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m emptyMessage

			m.Decode(rxmsg.Data)
			if s != nil {
				t.Errorf("expected nil, got %v", s)
			} else if !reflect.DeepEqual(&m, msg) {
				t.Errorf("expected %v, got %v", msg, m)
			}
			donech <- true
			return nil
		})
	transv.SubscribeMessage(
		&emptyMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m emptyMessage
			m.Decode(rxmsg.Data)
			if s != nil {
				t.Errorf("expected nil, got %v", s)
			} else if !reflect.DeepEqual(&m, msg) {
				t.Errorf("expected %v, got %v", msg, m)
			}
			transv.Post(&m, true)
			return nil
		})
	transc.Post(msg, true)
	<-donech

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestTransPostOnebyte(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	msg := &onebyteMessage{field: 'a'}
	donech := make(chan bool, 2)
	transc.SubscribeMessage(
		&onebyteMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m onebyteMessage
			m.Decode(rxmsg.Data)
			if s != nil {
				t.Errorf("expected nil, got %v", s)
			} else if !reflect.DeepEqual(&m, msg) {
				t.Errorf("expected %v, got %v", msg, m)
			}
			donech <- true
			return nil
		})
	transv.SubscribeMessage(
		&onebyteMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m onebyteMessage
			m.Decode(rxmsg.Data)
			if s != nil {
				t.Errorf("expected nil, got %v", s)
			} else if !reflect.DeepEqual(&m, msg) {
				t.Errorf("expected %v, got %v", msg, m)
			}
			transv.Post(&m, true)
			return nil
		})
	transc.Post(msg, true)
	<-donech

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestTransPostLarge(t *testing.T) {
	addr := <-testBindAddrs
	sconf := newsetts(TagOpaqueStart, TagOpaqueStart+10)
	sconf["buffersize"] = uint64(1024 * 1204)
	cconf := newsetts(TagOpaqueStart+11, TagOpaqueStart+20)
	cconf["buffersize"] = uint64(1024 * 1204)
	lis, serverch := newServersetts("server", addr, sconf)      // init server
	transc := newClientsetts("client", addr, cconf).Handshake() // init client
	transv := <-serverch
	// test
	msg := &largeMessage{}
	transc.SubscribeMessage(&largeMessage{}, nil)
	transv.SubscribeMessage(
		&largeMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m largeMessage

			m.Decode(rxmsg.Data)
			s.Response(&m, true)
			return nil
		})
	resp := &largeMessage{}
	if err := transc.Request(msg, true, resp); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(resp, msg) {
		t.Errorf("expected %v, got %v", msg, resp)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestTransRequest(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	msg := &testMessage{1234}
	transc.SubscribeMessage(&testMessage{}, nil)
	transv.SubscribeMessage(
		&testMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m testMessage

			m.Decode(rxmsg.Data)
			s.Response(&m, true)
			return nil
		})
	resp := &testMessage{}
	if err := transc.Request(msg, true, resp); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(resp, msg) {
		t.Errorf("expected %v, got %v", msg, resp)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestTransRequestEmpty(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	msg := &emptyMessage{}
	transc.SubscribeMessage(&emptyMessage{}, nil)
	transv.SubscribeMessage(
		&emptyMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m emptyMessage
			m.Decode(rxmsg.Data)
			s.Response(&m, true)
			return nil
		})
	resp := &emptyMessage{}
	if err := transc.Request(msg, true, resp); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(resp, msg) {
		t.Errorf("expected %v, got %v", msg, resp)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestTransRequestOnebyte(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	msg := &onebyteMessage{field: 'a'}
	transc.SubscribeMessage(&onebyteMessage{}, nil)
	transv.SubscribeMessage(
		&onebyteMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m onebyteMessage
			m.Decode(rxmsg.Data)
			s.Response(&m, true)
			return nil
		})
	resp := &onebyteMessage{}
	if err := transc.Request(msg, true, resp); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(resp, msg) {
		t.Errorf("expected %v, got %v", msg, resp)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestClientStream(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	start, n := uint64(1235), uint64(100)
	msg := &testMessage{1234}
	refch := make(chan Message)
	transc.SubscribeMessage(&testMessage{}, nil)
	transv.SubscribeMessage(
		&testMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m testMessage
			m.Decode(rxmsg.Data)
			refch <- &m
			return func(rxmsg BinMessage, ok bool) {
				if ok {
					var m testMessage
					m.Decode(rxmsg.Data)
					refch <- &m
				} else {
					close(refch)
				}
			}
		})
	stream, err := transc.Stream(msg, true, nil)
	if err != nil {
		t.Error(err)
	}
	for i := uint64(0); i < n; i++ {
		if err = stream.Stream(&testMessage{start + i}, true); err != nil {
			t.Error(err)
		}
	}
	if err = stream.Close(); err != nil {
		t.Error(err)
	}

	// validate
	i := uint64(1234)
	for msg := range refch {
		r := &testMessage{i}
		if !reflect.DeepEqual(msg, r) {
			t.Errorf("expected %#v, got %#v", r, msg)
		}
		i++
	}
	if ref := uint64(1234) + 100 + 1; i != ref {
		t.Errorf("expected %v, got %v", ref, i)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestClientStreamEmpty(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	n := uint64(1235)
	msg := &emptyMessage{}
	refch := make(chan Message, n+2)
	transc.SubscribeMessage(&emptyMessage{}, nil)
	transv.SubscribeMessage(
		&emptyMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m emptyMessage
			m.Decode(rxmsg.Data)
			refch <- &m
			return func(rxmsg BinMessage, ok bool) {
				if ok {
					var m emptyMessage
					m.Decode(rxmsg.Data)
					refch <- &m
				} else {
					close(refch)
				}
			}
		})
	stream, err := transc.Stream(msg, true, nil)
	if err != nil {
		t.Error(err)
	}
	for i := uint64(0); i < n; i++ {
		if err = stream.Stream(&emptyMessage{}, true); err != nil {
			t.Error(err)
		}
	}
	if err = stream.Close(); err != nil {
		t.Error(err)
	}
	time.Sleep(100 * time.Millisecond)

	// validate
	if uint64(len(refch)) != n+1 {
		t.Errorf("expected %v, got %v", n+1, len(refch))
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestClientStreamOnebyte(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	n := uint64(1235)
	msg := &onebyteMessage{field: 'a'}
	refch := make(chan Message, n+2)
	transc.SubscribeMessage(&onebyteMessage{}, nil)
	transv.SubscribeMessage(
		&onebyteMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m onebyteMessage
			m.Decode(rxmsg.Data)
			refch <- &m
			return func(rxmsg BinMessage, ok bool) {
				if ok {
					var m onebyteMessage
					m.Decode(rxmsg.Data)
					refch <- &m
				} else {
					close(refch)
				}
			}
		})
	stream, err := transc.Stream(msg, true, nil)
	if err != nil {
		t.Error(err)
	}
	for i := uint64(0); i < n; i++ {
		err = stream.Stream(&onebyteMessage{field: 'a'}, true)
		if err != nil {
			t.Error(err)
		}
	}
	if err = stream.Close(); err != nil {
		t.Error(err)
	}
	time.Sleep(100 * time.Millisecond)

	// validate
	if uint64(len(refch)) != n+1 {
		t.Errorf("expected %v, got %v", n+1, len(refch))
	}
	for checkmsg := range refch {
		if reflect.DeepEqual(checkmsg, msg) == false {
			t.Errorf("expected %v, got %v", msg, checkmsg)
		}
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestServerStream(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	// test
	start, n := uint64(1235), uint64(100)
	msg := &testMessage{1234}
	refch := make(chan Message)
	transc.SubscribeMessage(
		&testMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m testMessage
			m.Decode(rxmsg.Data)
			refch <- &m
			return func(rxmsg BinMessage, ok bool) {
				if ok {
					var m testMessage
					m.Decode(rxmsg.Data)
					refch <- &m
				} else {
					close(refch)
				}
			}
		})
	transv.SubscribeMessage(&testMessage{}, nil)
	stream, err := transv.Stream(msg, true, nil)
	if err != nil {
		t.Error(err)
	}
	for i := uint64(0); i < n; i++ {
		if err = stream.Stream(&testMessage{start + i}, true); err != nil {
			t.Error(err)
		}
	}
	if err = stream.Close(); err != nil {
		t.Error(err)
	}

	// validate
	i := uint64(1234)
	for msg := range refch {
		r := &testMessage{i}
		if !reflect.DeepEqual(msg, r) {
			t.Errorf("expected %#v, got %#v", r, msg)
		}
		i++
	}
	if ref := uint64(1234) + 100 + 1; i != ref {
		t.Errorf("expected %v, got %v", ref, i)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestTransGzip(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "gzip")      // init server
	transc := newClient("client", addr, "gzip").Handshake() // init client
	transv := <-serverch
	// test
	msg := &testMessage{1234}
	transc.SubscribeMessage(&testMessage{}, nil)
	transv.SubscribeMessage(
		&testMessage{},
		func(s *Stream, rxmsg BinMessage) StreamCallback {
			var m testMessage
			m.Decode(rxmsg.Data)
			s.Response(&m, true)
			return nil
		})
	resp := &testMessage{}
	if err := transc.Request(msg, true, resp); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(resp, msg) {
		t.Errorf("expected %v, got %v", msg, resp)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func TestJunkRx(t *testing.T) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "gzip")      // init server
	transc := newClient("client", addr, "gzip").Handshake() // init client
	transv := <-serverch
	// test
	if _, err := transc.conn.Write([]byte("junk")); err != nil {
		t.Error(err)
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

func BenchmarkTransStats(b *testing.B) {
	addr := <-testBindAddrs
	lis, serverch := newServer("server", addr, "")      // init server
	transc := newClient("client", addr, "").Handshake() // init client
	transv := <-serverch
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		transc.Stat()
	}

	time.Sleep(100 * time.Millisecond)

	lis.Close()
	transc.Close()
	transv.Close()
}

//---- test fixture with client and server.

func newsetts(start, end uint64) s.Settings {
	return DefaultSettings(int64(start), int64(end))
}

func newServer(name, addr, tags string) (*net.TCPListener, chan *Transport) {
	setts := newsetts(TagOpaqueStart, TagOpaqueStart+10)
	setts["tags"] = tags
	return newServersetts(name, addr, setts)
}

func newServersetts(
	name, addr string, setts s.Settings) (*net.TCPListener, chan *Transport) {

	la, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		panic(err)
	}
	lis, err := net.ListenTCP("tcp", la)
	if err != nil {
		panic(fmt.Errorf("listen failed %v", err))
	}
	if fd, err := lis.File(); err == nil {
		syscall.SetsockoptInt(
			int(fd.Fd()), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	} else {
		panic(err)
	}

	ch := make(chan *Transport, 10)
	go func() {
		if conn, err := lis.Accept(); err == nil {
			ver := testVersion(1)
			trans, err := NewTransport(name, conn, &ver, setts)
			if err != nil {
				panic("NewTransport server failed")
			}
			ch <- trans.Handshake()
		}
	}()
	return lis, ch
}

func newClient(name, addr, tags string) *Transport {
	setts := newsetts(TagOpaqueStart+11, TagOpaqueStart+20)
	setts["tags"] = tags
	return newClientsetts(name, addr, setts)
}

func newClientsetts(name, addr string, setts s.Settings) *Transport {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}
	ver := testVersion(1)
	trans, err := NewTransport(name, conn, &ver, setts)
	if err != nil {
		panic(err)
	}
	return trans
}

type testConnection struct {
	roff  int
	woff  int
	buf   []byte
	read  bool
	mu    sync.Mutex
	laddr netAddr
	raddr netAddr
}

func newTestConnection(l, r string, buf []byte, read bool) *testConnection {
	tconn := &testConnection{
		laddr: netAddr(l),
		raddr: netAddr(r),
		read:  read,
	}
	if tconn.buf = buf; buf == nil {
		tconn.buf = make([]byte, 100000)
	}
	return tconn
}

func (tc *testConnection) Write(b []byte) (n int, err error) {
	do := func() (int, error) {
		tc.mu.Lock()
		defer tc.mu.Unlock()
		newoff := tc.woff + len(b)
		copy(tc.buf[tc.woff:newoff], b)
		tc.woff = newoff
		return len(b), nil
	}
	n, err = do()
	//fmt.Println("write ...", n, err)
	return
}

func (tc *testConnection) Read(b []byte) (n int, err error) {
	do := func() (int, error) {
		tc.mu.Lock()
		defer tc.mu.Unlock()
		if newoff := tc.roff + len(b); newoff <= tc.woff {
			copy(b, tc.buf[tc.roff:newoff])
			tc.roff = newoff
			return len(b), nil
		}
		return 0, nil
	}
	for err == nil && n == 0 {
		if tc.read {
			n, err = do()
		}
		time.Sleep(10 * time.Millisecond)
	}
	//fmt.Println("read ...", n, err)
	return n, err
}

func (tc *testConnection) LocalAddr() net.Addr {
	return tc.laddr
}

func (tc *testConnection) RemoteAddr() net.Addr {
	return tc.raddr
}

func (tc *testConnection) reset() *testConnection {
	tc.woff, tc.roff = 0, 0
	return tc
}

func (tc *testConnection) Close() error {
	return nil
}

type netAddr string

func (addr netAddr) Network() string {
	return "tcp"
}

func (addr netAddr) String() string {
	return string(addr)
}

type testVersion int

func (v *testVersion) Less(ver Version) bool {
	return (*v) < (*ver.(*testVersion))
}

func (v *testVersion) Equal(ver Version) bool {
	return (*v) == (*ver.(*testVersion))
}

func (v *testVersion) String() string {
	return fmt.Sprintf("%v", int(*v))
}

func (v *testVersion) Encode(out []byte) []byte {
	out = fixbuffer(out, 32)
	n := valuint642cbor(uint64(*v), out)
	return out[:n]
}

func (v *testVersion) Decode(in []byte) int64 {
	ln, n := cborItemLength(in)
	*v = testVersion(ln)
	return int64(n)
}

func (v *testVersion) Size() int64 {
	return 9
}

func printStats(counts map[string]uint64) {
	keys := []string{}
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Sort(sort.StringSlice(keys))
	s := []string{}
	for _, key := range keys {
		s = append(s, fmt.Sprintf("%v:%v", key, counts[key]))
	}
	fmt.Println(strings.Join(s, ", "))
}

func verify(counts map[string]uint64, args ...interface{}) bool {
	check := func(keys []string, val uint64) bool {
		for _, k := range keys {
			if counts[k] != val {
				return false
			}
		}
		return true
	}
	keys := []string{}
	for _, arg := range args {
		if k, ok := arg.(string); ok {
			keys = append(keys, k)
		} else if i, ok := arg.(int); ok {
			if check(keys, uint64(i)) == false {
				return false
			}
			keys = []string{}
		} else if i64, ok := arg.(uint64); ok {
			if check(keys, i64) == false {
				return false
			}
			keys = []string{}
		}
	}
	if len(keys) > 0 {
		x := counts[keys[0]]
		for _, k := range keys[1:] {
			if x != counts[k] {
				return false
			}
		}
	}
	return true
}

var testBindAddrs chan string

func init() {
	testBindAddrs = make(chan string, 1000)
	for i := 0; i < cap(testBindAddrs); i++ {
		testBindAddrs <- fmt.Sprintf("127.0.0.1:%v", 9100+i)
	}
	//runtime.GOMAXPROCS(runtime.NumCPU())

	logsetts := map[string]interface{}{"log.level": "warn", "log.file": ""}
	log.SetLogger(nil, logsetts)
}
