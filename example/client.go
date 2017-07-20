package main

import "sync"
import "log"
import "time"
import "net"
import "fmt"
import "strconv"
import "math/rand"

import gf "github.com/prataprc/gofast"

var av = &averageInt64{}

func client() {
	doTransport()
}

func doTransport() {
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
			opqstart, opqend := 16000, 16000+options.routines
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
	start := time.Now()
	wg.Wait()
	elapsed := time.Since(start)
	for _, trans := range transs {
		trans.Close()
	}
	printCounts(addCounts(transs...))
	fmsg := "request stats: n:%v mean:%v var:%v sd:%v\n"
	n, m := av.samples(), time.Duration(av.mean())
	v, s := time.Duration(av.variance()), time.Duration(av.sd())
	log.Printf(fmsg, n, m, v, s)
	fmsg = "elapsed time for %v messages: %v\n"
	log.Printf(fmsg, options.count*options.conns, elapsed)
}

func worker(t *gf.Transport, doch chan int, wg *sync.WaitGroup) {
	key := make([]byte, 25)
	msgpost := &msgPost{value: make([]byte, options.payload)}
	for i := range msgpost.value {
		msgpost.value[i] = 'a'
	}
	key = strconv.AppendInt(key[:0], int64(rand.Intn(10000000)), 16)
	key = append(key, '-')
	lnkey := len(key)

	n := 1
	for doit := range doch {
		since := time.Now()
		switch doit {
		case sendpost:
			k := strconv.AppendInt(key[lnkey:], int64(n), 10)
			msgpost.key = key[:lnkey+len(k)]
			if err := t.Post(msgpost, false); err != nil {
				fmt.Printf("%v\n", err)
				panic("exit")
			}
		}
		av.add(int64(time.Since(since)))
		n++
	}
	if _, err := t.Whoami(); err != nil {
		log.Fatal(err)
	}
	wg.Done()
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
