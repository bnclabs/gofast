package main

import "time"
import "sync"
import "log"
import "net"
import "net/http"
import "fmt"

import gf "github.com/prataprc/gofast"
import _ "github.com/prataprc/gofast/http"

var mu sync.Mutex
var transs = make([]*gf.Transport, 0, 100)

func server() {
	// start http server
	go func() {
		log.Println(http.ListenAndServe(":8080", nil))
	}()
	// listen
	lis, err := net.Listen("tcp", options.addr)
	if err != nil {
		panic(fmt.Errorf("listen failed %v", err))
	}
	fmt.Printf("listening on %v\n", options.addr)
	// wait for connections
	runserver(lis)
}

func runserver(lis net.Listener) {
	ver := testVersion(1)

	opqend := 1000 + options.routines
	config := newconfig(1000, opqend)
	config["tags"] = options.tags
	config["batchsize"] = options.batchsize
	conncount := 0
	for {
		if conn, err := lis.Accept(); err == nil {
			name := fmt.Sprintf("server-%v", conncount)
			conncount += 1
			fmt.Println("new transport", conn.RemoteAddr(), conn.LocalAddr())
			trans, err := gf.NewTransport(name, conn, &ver, config)
			if err != nil {
				panic("NewTransport server failed")
			}
			mu.Lock()
			transs = append(transs, trans)
			mu.Unlock()
			go func(trans *gf.Transport) {
				trans.FlushPeriod(options.flushtick * time.Millisecond)
				trans.SendHeartbeat(1 * time.Second)
				trans.SubscribeMessage(
					&msgPost{},
					func(s *gf.Stream, msg gf.BinMessage) gf.StreamCallback {
						// Fill up your handler code here.
						return nil
					})
				trans.Handshake()
				//tick := time.Tick(1 * time.Second)
				//for {
				//	<-tick
				//	if options.log == "debug" {
				//		printCounts(trans.Stat())
				//	}
				//}
			}(trans)
		}
	}
}
