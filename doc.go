// Package gofast implements a high performance symmetric protocol for on the
// wire data transport.
//
// opaque-space, is range of uint64 values reserved for tagging packets. They
// shall be supplied via settings while instantiating the transport.
//
// messages, are golang objects implementing the Message{} interface. Message
// objects need to be subscribed with transport before they are exchanged over
// the transport. It is also expected that distributed systems must
// pre-define messages and their Ids.
//
// message ids, need to be unique for every type of message transfered over
// using gofast protocol, following id range is reserved for internal use:
//
//		0x00 - 0x0F -- reserved messages ids.
//
// transport instantiation steps:
//
//		setts := Settings{"log.level": "info", "log.file": logfile}
//		log := SetLogger(nil /* use-default-logging */, setts)
//		t := NewTransport(conn, &ver, nil, settings)
//		t.SubscribeMessage(&msg1, handler1) // subscribe message
//		t.SubscribeMessage(&msg2, handler2) // subscribe another message
//		t.Handshake()
//		t.FlushPeriod(tm)                   // optional
//		t.SendHeartbeat(tm)                 // optional
package gofast
