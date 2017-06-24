// Package gofast implement high performance symmetric protocol for on the
// wire data transport. A single socket connection can be used to create a
// gofast.Transport, and once created application can concurrently post,
// request, stream messages on the same transport (same socket). Internally
// gofast library maintain a global collection of all active gofast.Transport
// that are created during the lifetime of application.
//
// Opaque-space, is range of uint64 values reserved for tagging packets. They
// shall be supplied via settings while instantiating the transport. Note that
// Opaque-space can be less than 256.
//
// Messages are golang objects implementing the Message{} interface. Message
// objects need to be subscribed with transport before they are exchanged over
// the transport. It is also expected that applications using gofast should
// pre-define messages and their Ids.
//
// message ids, need to be unique for every type of message transfered
// using gofast protocol, following id range is reserved for internal use:
//
//		0x1000 - 0x100F -- reserved messages ids.
//
// transport instantiation steps:
//
//		setts := gosettings.Settings{
//				"log.level": "info", "log.file": logfile,
//		}
//		golog.SetLogger(nil /* use-default-logging */, setts)
//
//		t := NewTransport(conn, &ver, nil, settings)
//		t.SubscribeMessage(&msg1, handler1) // subscribe message
//		t.SubscribeMessage(&msg2, handler2) // subscribe another message
//		t.Handshake()
//		t.FlushPeriod(tm)                   // optional
//		t.SendHeartbeat(tm)                 // optional
//
// If your application is using a custom logger, implement golog.Logger{}
// interface on your custom logger and supply that as first argument
// to SetLogger().
package gofast
