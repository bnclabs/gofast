package main

import "log"
import "time"
import "os"
import "fmt"
import "flag"
import "encoding/json"

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
	if len(flag.Args()) == 0 {
		fmt.Println("command missing, either `post` or `request`")
		os.Exit(1)
	}
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
	server.SetPostHandler(func(request interface{}) {})

	flags := gofast.TransportFlag(gofast.EncodingBinary)
	ch := make(chan bool, options.par)
	for i := 0; i < options.par; i++ {
		go doPost(flags, clientConfig, options.count, ch)
	}
	for i := 0; i < options.par; i++ {
		log.Println("post completed", i, <-ch)
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
		client.Post(flags, 1, []byte("hello world"))
	}
	ch <- true
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
	server.SetRequestHandlerFor(
		1,
		func(req interface{}, send gofast.ResponseSender) {
			go func() {
				var r []interface{}
				// unmarshal request
				if err := json.Unmarshal(req.([]byte), &r); err != nil {
					log.Fatal(err)
				}
				// construct response
				data, err := json.Marshal([]interface{}{r[0], r[1], "response"})
				if err != nil {
					log.Fatal(err)
				}
				// introduce a delay
				if options.latency > 0 {
					time.Sleep(time.Duration(options.latency) * time.Millisecond)
				}
				// and send back response
				send(2, data, true)
				donech <- true
			}()
		})

	client, err := gofast.NewClient(options.host, clientConfig, nil)
	if err != nil {
		log.Fatal(err)
	}
	client.SetEncoder(gofast.EncodingBinary, nil)
	client.Start()

	for i := 0; i < options.par; i++ {
		go func() {
			k := i
			for j := 0; j < options.count; j++ {
				// construct request
				data, err := json.Marshal([]interface{}{k, j, "request"})
				if err != nil {
					log.Fatal(err)
				}
				// make request
				resp, err := client.Request(flags, 1, data)
				if err != nil {
					log.Fatal(err)
				}
				// construct response
				var r []interface{}
				if err := json.Unmarshal(resp.([]byte), &r); err != nil {
					log.Fatal(err)
				}
				// verify
				if r[0].(float64) != float64(k) {
					log.Fatalf("expected %v, got %v\n", []interface{}{k, j}, r)
				} else if r[1].(float64) != float64(j) {
					log.Fatalf("expected %v, got %v\n", []interface{}{k, j}, r)
				} else if r[2].(string) != "response" {
					log.Fatalf("expected %v, got %v\n", []interface{}{k, j}, r)
				}
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
	log.Println("AllCompleted ", count)
	client.Close()
}
