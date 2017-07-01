package main

import "time"
import "sync"
import "io"
import "os"
import "log"
import "bufio"
import "net"
import "net/http"
import "fmt"
import "math/rand"
import "runtime/pprof"

import gf "github.com/prataprc/gofast"
import _ "github.com/prataprc/gofast/http"

var mu sync.Mutex
var transs = make([]*gf.Transport, 0, 100)

func server() {
	// start http server
	go func() {
		log.Println(http.ListenAndServe(":8080", nil))
	}()

	// start cpu profile.
	if options.profile {
		fd := startCpuProfile("server.pprof")
		defer fd.Close()
		defer pprof.StopCPUProfile()
	}

	lis, err := net.Listen("tcp", options.addr)
	if err != nil {
		panic(fmt.Errorf("listen failed %v", err))
	}
	fmt.Printf("listening on %v\n", options.addr)

	start := time.Now()
	go runserver(lis)

	fmt.Println("Press CTRL-D to exit")
	reader := bufio.NewReader(os.Stdin)
	_, err = reader.ReadString('\n')
	for err != io.EOF {
		_, err = reader.ReadString('\n')
	}
	fmt.Println("server exited")
	mu.Lock()
	elapsed := time.Since(start)
	printCounts(addCounts(transs...), elapsed, nil)
	mu.Unlock()

	// take memory profile.
	if options.profile {
		doMemProfile("server.mprof")
	}
}

func runserver(lis net.Listener) {
	ver := testVersion(1)

	opqend := (1000 + (options.routines * 3))
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
						return nil
					})
				trans.SubscribeMessage(
					&msgReqsp{},
					func(s *gf.Stream, msg gf.BinMessage) gf.StreamCallback {
						var rmsg msgReqsp
						rmsg.Decode(msg.Data)
						if err := s.Response(&rmsg, true); err != nil {
							log.Fatal(err)
						}
						return nil
					})
				trans.SubscribeMessage(
					&msgStreamRx{},
					func(s *gf.Stream, msg gf.BinMessage) gf.StreamCallback {
						closeat := rand.Intn(options.stream)
						return func(rxstrmsg gf.BinMessage, ok bool) {
							if options.do == "verify" {
								if closeat == 0 {
									s.Close()
								}
								closeat--
							}
							if ok == false {
								s.Close()
							}
						}
					})
				trans.SubscribeMessage(
					&msgStreamTx{},
					func(s *gf.Stream, msg gf.BinMessage) gf.StreamCallback {
						go func() {
							tmsg := &msgStreamTx{
								data: make([]byte, options.payload),
							}
							for i := 0; i < options.payload; i++ {
								tmsg.data[i] = 'a'
							}
							count := options.stream
							if options.do == "verify" {
								count = rand.Intn(options.stream)
							}
							for i := 0; i < count; i++ {
								if err := s.Stream(tmsg, true); err != nil {
									log.Printf("error stream: %v\n", err)
								}
							}
							s.Close()
						}()
						return nil
					})

				trans.Handshake()
				tick := time.Tick(1 * time.Second)
				for {
					<-tick
					if options.log == "debug" {
						printCounts(trans.Stat(), 0, nil)
					}
				}
			}(trans)
		}
	}
}
