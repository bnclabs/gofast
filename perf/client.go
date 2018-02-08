package main

import "sync"
import "log"
import "time"
import "net"
import "fmt"
import "math/rand"
import "runtime/pprof"

import gf "github.com/bnclabs/gofast"

var av = &averageInt64{}

func client() {
	// start cpu profile, always enabled.
	if options.profile {
		fd := startCpuProfile("client.pprof")
		defer fd.Close()
		defer pprof.StopCPUProfile()
	}

	start := time.Now()
	counts := doTransport()
	elapsed := time.Since(start)
	printCounts(counts, elapsed, av)

	// take memory profile.
	if options.profile {
		doMemProfile("client.mprof")
	}
}

const (
	sendpost int = iota + 1
	sendreq
	sendstream
	rcvstream
)

func postit() int {
	return sendpost
}

func requestit() int {
	return sendreq
}

func streamtx() int {
	return sendstream
}

func streamrx() int {
	return rcvstream
}

func randomit() int {
	now := time.Now().UnixNano()
	if (now % 100) == 0 {
		if now%200 == 0 {
			return sendstream
		}
		return rcvstream
	}
	if now%2 == 0 {
		return sendreq
	}
	return sendpost
}

func doTransport() map[string]uint64 {
	var wg sync.WaitGroup

	ver := testVersion(1)
	transs := make([]*gf.Transport, 0, 16)

	var doit func() int

	switch options.do {
	case "post":
		doit = postit
	case "request":
		doit = requestit
	case "streamrx":
		doit = streamrx
	case "streamtx":
		doit = streamtx
	case "verify":
		doit = randomit
	}

	for i := 0; i < options.conns; i++ {
		wg.Add(options.routines)
		go func(i int) {
			opqstart, opqend := 160000, (160000 + (options.routines * 3))
			name := fmt.Sprintf("client-%v", i)
			config := newconfig(opqstart, opqend)
			config["tags"] = options.tags
			config["batchsize"] = options.batchsize
			conn, err := net.Dial("tcp", options.addr)
			if err != nil {
				panic(err)
			}
			trans, err := gf.NewTransport(name, conn, &ver, config)
			if err != nil {
				panic(err)
			}
			trans.FlushPeriod(options.flushtick * time.Millisecond)
			trans.SendHeartbeat(1 * time.Second)
			trans.SubscribeMessage(&msgPost{}, nil)
			trans.SubscribeMessage(&msgReqsp{}, nil)
			trans.SubscribeMessage(&msgStreamRx{}, nil)
			trans.SubscribeMessage(&msgStreamTx{}, nil)
			if err := trans.Handshake(); err != nil {
				panic(err)
			}

			transs = append(transs, trans)

			doers := make([]chan int, 0, 100)
			for j := 0; j < options.routines; j++ {
				doch := make(chan int, options.count/options.routines)
				go worker(trans, doch, &wg)
				doers = append(doers, doch)
			}

			go func(trans *gf.Transport, count int, doers []chan int) {
				numdoers := len(doers)
				for k := 0; k < count; k++ {
					doers[k%numdoers] <- doit()
				}
				for _, doch := range doers {
					close(doch)
				}
			}(trans, options.count, doers)
		}(i)
	}
	wg.Wait()
	for _, trans := range transs {
		trans.Close()
	}
	return addCounts(transs...)
}

func worker(t *gf.Transport, doch chan int, wg *sync.WaitGroup) {
	msgpost := &msgPost{data: make([]byte, options.payload)}
	for i := 0; i < options.payload; i++ {
		msgpost.data[i] = 'a'
	}
	msgreq := &msgReqsp{data: make([]byte, options.payload)}
	for i := 0; i < options.payload; i++ {
		msgreq.data[i] = 'a'
	}
	msgstrmrx := &msgStreamRx{data: make([]byte, options.payload)}
	for i := 0; i < options.payload; i++ {
		msgstrmrx.data[i] = 'a'
	}
	msgstrmtx := &msgStreamTx{data: make([]byte, options.payload)}
	for i := 0; i < options.payload; i++ {
		msgstrmtx.data[i] = 'a'
	}

	for doit := range doch {
		since := time.Now()
		switch doit {
		case sendpost:
			if err := t.Post(msgpost, false); err != nil {
				fmt.Printf("%v\n", err)
				panic("exit")
			}

		case sendreq:
			var rmsg msgReqsp
			if err := t.Request(msgreq, false, &rmsg); err != nil {
				fmt.Printf("%v\n", err)
				panic("exit")
			}

		case sendstream:
			count := options.stream
			if options.do == "verify" {
				count = rand.Intn(options.stream)
			}
			doStreamTx(t, count, msgstrmrx)

		case rcvstream:
			doStreamRx(t, 0, msgstrmtx)
		}
		av.add(int64(time.Since(since)))
	}
	if _, err := t.Whoami(); err != nil {
		log.Fatal(err)
	}
	wg.Done()
}

func doStreamTx(trans *gf.Transport, count int, msg gf.Message) {
	stream, err := trans.Stream(
		msg, false, func(rxstrmsg gf.BinMessage, ok bool) {})
	if err != nil {
		log.Fatal(err)
	}
	for j := 0; j < count; j++ {
		if err := stream.Stream(msg, false); err != nil {
			log.Printf("error stream: %v\n", err)
			break
		}
	}
	stream.Close()
	if options.do == "verify" {
		time.Sleep(1 * time.Second)
	}
}

func doStreamRx(trans *gf.Transport, count int, msg gf.Message) {
	closeat, donech := rand.Intn(options.stream), make(chan struct{})
	stream, err := trans.Stream(
		msg, false, func(rxstrmsg gf.BinMessage, ok bool) {
			if options.do == "verify" {
				if closeat == 0 {
					close(donech)
				}
				closeat--
			}
		})
	if err != nil {
		log.Fatal(err)
	}
	<-donech
	stream.Close()
	if options.do == "verify" {
		time.Sleep(1 * time.Second)
	}
}
