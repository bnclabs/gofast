package main

import "flag"
import "fmt"
import "log"
import "time"
import "runtime"
import "net/http"
import _ "net/http/pprof"

import golog "github.com/bnclabs/golog"
import s "github.com/bnclabs/gosettings"

var options struct {
	// general options.
	cpu       int
	tags      string
	addr      string
	log       string
	stream    int
	batchsize int
	profile   bool

	// client specific options.
	client     bool
	do         string
	count      int
	routines   int
	conns      int
	payload    int
	buffersize int
	flushtick  time.Duration

	// server specific options.
	//
	server bool
}

func argParse() {
	var flushtick int

	// generic options
	flag.IntVar(&options.cpu, "cpu", runtime.NumCPU(),
		"GOMAXPROCS")
	flag.StringVar(&options.tags, "tags", "",
		"server and client option, comma separated list of tags")
	flag.StringVar(&options.addr, "addr", "127.0.0.1:9998",
		"server address")
	flag.IntVar(&options.routines, "routines", 1,
		"number of concurrent routines per connection")
	flag.IntVar(&options.stream, "stream", 1000000,
		"number of stream messages per stream request")
	flag.StringVar(&options.do, "do", "post",
		"post / request / stream benchmark to do")
	flag.IntVar(&options.payload, "payload", 512,
		"payload size to ping pong.")
	flag.IntVar(&options.buffersize, "buffersize", 2048,
		"buffersize for batching.")
	flag.IntVar(&options.batchsize, "batchsize", 1,
		"no. of messages to batch")
	flag.BoolVar(&options.profile, "profile", false,
		"take cpuprofile and memprofile")
	flag.IntVar(&flushtick, "flushtick", 10,
		"flush period in milliseconds.")
	flag.StringVar(&options.log, "log", "error",
		"log level")

	options.flushtick = time.Duration(flushtick) * time.Millisecond

	// client specific options
	flag.BoolVar(&options.client, "c", false,
		"start in client mode")
	flag.IntVar(&options.conns, "conns", 1,
		"number of connections to use")
	flag.IntVar(&options.count, "count", 1,
		"number of requests per routine")

	// server specific options
	flag.BoolVar(&options.server, "s", false,
		"start in server mode")

	flag.Parse()
}

func main() {
	argParse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	golog.SetLogger(nil, s.Settings{"log.level": "warn", "log.file": ""})

	if options.server {
		go func() {
			log.Println(http.ListenAndServe("localhost:6062", nil))
		}()
		server()

	} else if options.client {
		go func() {
			log.Println(http.ListenAndServe("localhost:6063", nil))
		}()
		client()
	} else {
		fmt.Println("ERROR: either start in server mode or client mode")
	}
}
