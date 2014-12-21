package main

import "log"
import "time"
import "flag"
import "github.com/prataprc/gofast"

var options struct {
	count int
	par   int
}

func argParse() {
	flag.IntVar(&options.count, "count", 1000,
		"number of posts per connection")
	flag.IntVar(&options.par, "par", 1,
		"number of parallel clients (aka connections)")
	flag.Parse()
	return
}

func main() {
	argParse()
	flags := gofast.TransportFlag(gofast.EncodingBinary)
	clientConfig := map[string]interface{}{
		"maxPayload":     1024 * 1024,
		"writeDeadline":  4 * 1000, // 4 seconds
		"muxChanSize":    100000,
		"streamChanSize": 10000,
	}
	serverConfig := map[string]interface{}{
		"maxPayload":     1024 * 1024,
		"writeDeadline":  4 * 1000, // 4 seconds
		"reqChanSize":    1000,
		"streamChanSize": 10000,
	}
	host := ":9999"

	// start server
	server, err := gofast.NewServer(host, serverConfig, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	server.SetEncoder(gofast.EncodingBinary, nil)
	server.SetPostHandler(func(opaque uint32, request interface{}) {})

	ch := make(chan bool, options.par)
	for i := 0; i < options.par; i++ {
		go doPost(host, flags, clientConfig, options.count, ch)
	}
	for i := 0; i < options.par; i++ {
		log.Println(<-ch)
	}
}

func doPost(
	host string, flags gofast.TransportFlag, config map[string]interface{},
	count int, ch chan bool) {

	client, err := gofast.NewClient(host, config, nil)
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
