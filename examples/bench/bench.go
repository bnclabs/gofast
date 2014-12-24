package main

import "log"
import "time"
import "flag"
import "github.com/prataprc/gofast"

var options struct {
	count   int
	par     int
	latency int
	host    string // not via cmdline
	debug   bool
	trace   bool
}

func argParse() string {
	flag.IntVar(&options.count, "count", 1000,
		"number of posts per connection")
	flag.IntVar(&options.par, "par", 1,
		"number of parallel clients (aka connections)")
	flag.IntVar(&options.latency, "latency", 0,
		"latency to serve each request")
	flag.BoolVar(&options.debug, "debug", false,
		"enable debug level logging")
	flag.BoolVar(&options.trace, "trace", false,
		"enable trace level logging")
	flag.Parse()
	options.host = ":9999"
	return flag.Args()[0]
}

var clientConfig = map[string]interface{}{
	"maxPayload":     1024 * 1024,
	"writeDeadline":  4 * 1000, // 4 seconds
	"muxChanSize":    100000,
	"streamChanSize": 10000,
}
var serverConfig = map[string]interface{}{
	"maxPayload":     1024 * 1024,
	"writeDeadline":  4 * 1000, // 4 seconds
	"reqChanSize":    1000,
	"streamChanSize": 10000,
}

func main() {
	command := argParse()
	if options.trace {
		gofast.SetLogLevel(gofast.LogLevelTrace)
	} else if options.debug {
		gofast.SetLogLevel(gofast.LogLevelDebug)
	}
	switch command {
	case "post":
		benchPost()
	case "request":
		benchRequest()
	}
}

func benchPost() {
	// start server
	server, err := gofast.NewServer(options.host, serverConfig, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	server.SetEncoder(gofast.EncodingBinary, nil)
	server.SetPostHandler(func(opaque uint32, request interface{}) {})

	flags := gofast.TransportFlag(gofast.EncodingBinary)
	ch := make(chan bool, options.par)
	for i := 0; i < options.par; i++ {
		go doPost(flags, clientConfig, options.count, ch)
	}
	for i := 0; i < options.par; i++ {
		log.Println(<-ch)
	}
}

func doPost(
	flags gofast.TransportFlag, config map[string]interface{},
	count int, ch chan bool) {

	client, err := gofast.NewClient(options.host, config, nil)
	if err != nil {
		log.Fatal(err)
	}
	client.SetEncoder(gofast.EncodingBinary, nil)
	client.Start()
	for i := 0; i < count; i++ {
		client.Post(flags, []byte("hello world"))
	}
	ch <- true
	time.Sleep(1 * time.Millisecond)
}

func benchRequest() {
	// start server
	server, err := gofast.NewServer(options.host, serverConfig, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	flags := gofast.TransportFlag(gofast.EncodingBinary)
	donech := make(chan bool, options.par)
	server.SetEncoder(gofast.EncodingBinary, nil)
	for i := 0; i < options.par; i++ {
		server.SetRequestHandlerFor(
			0x80000000+uint32(i),
			func(opaque uint32, req interface{}, respch chan []interface{}) {
				go func() {
					if options.latency > 0 {
						time.Sleep(time.Duration(options.latency) * time.Millisecond)
					}
					respch <- []interface{}{opaque, []byte("response1")}
					donech <- true
				}()
			})
	}

	client, err := gofast.NewClient(options.host, clientConfig, nil)
	if err != nil {
		log.Fatal(err)
	}
	client.SetEncoder(gofast.EncodingBinary, nil)
	client.Start()

	for i := 0; i < options.par; i++ {
		opaque := 0x80000000 + uint32(i)
		go func() {
			for i := 0; i < options.count; i++ {
				client.RequestWith(opaque, flags, []byte("request1"))
			}
		}()
	}
	count := 0
	tick := time.Tick(1 * time.Second)
	for i := options.par * options.count; i > 0; {
		select {
		case <-donech:
			i--
			count++
		case <-tick:
			log.Println("Completed ", count)
		}
	}
	client.Close()
}
