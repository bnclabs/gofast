Performance benchmark gofast
============================

**Following benchmark is done using quadcore (8 thread) macbook pro with
8 GB of RAM.**

Program under perf/ can be used to benchmark POST, REQUEST and STREAM messages
using gofast. On the client side, there is no logic other than filling up a
random payload and sending it to remote. Likewise server code is kept to
bare-minimum.

```bash
$ cd perf/
$ go build
```

**Start the server**

To start the `perf` program as server use the `-s` switch, supply port address
via `-addr` argument.

```bash
$ ./perf -s -addr :9900
```

**POST Benchmark**

To start the `perf` program as client use the `-c` switch, supply the server
address via `-addr` argument. Do sufficiently large number of POST so that
we can time the entire operation, in this case we will POST 1-million messages
to server.

```bash
time ./perf -c -addr localhost:9900 -do post -count 1000000

Latency Average: 17.76µs
Throughput: 55556 /second

stats { "n_dropped":0, "n_flushes":1000021, "n_mdrops":0, "n_rx":21, "n_rxbeats":18, "n_rxbyte":669, "n_rxfin":0, "n_rxpost":18, "n_rxreq":1, "n_rxresp":2, "n_rxstart":0, "n_rxstream":0, "n_tx":1000021, "n_txbyte":546000669, "n_txfin":0, "n_txpost":1000018, "n_txreq":2, "n_txresp":1, "n_txstart":0, "n_txstream":0 }
request stats: n:1000000 mean:17.76µs var:1m18.075045088s sd:279.419µs
./perf -c -addr localhost:9900 -do post -count 1000000  8.26s user 28.19s system 202% cpu 18.035 total
```

Client program completes in **17.8** seconds and dumps the final statistics
for all transport. For rest of the experiments we are going to ignore the raw
stats map and focus only on Latency and Throughput. In the above run, we were
using default values for connections (1-connection), batchsize (1), and
routines (1).

Let us increase the concurrency / batchsize:

```bash
time ./perf -c -addr localhost:9900 -do post -batchsize 100 -routines 100 -count 1000000

Latency Average: 616.84µs
Throughput: 166667 /second
```

Throughput increases, but latency also increases. Are they related ? Let us
decrease the batchsize back to 1.

```bash
time ./perf -c -addr localhost:9900 -do post -batchsize 1 -routines 100 -count 1000000

Latency Average: 2.091543ms
Throughput: 50001 /second
```

`batchsize` shows definite improvement with concurrency routines. Let us
increase the number of connections.

```bash
time ./perf -c -addr localhost:9900 -do post -conns 8 -batchsize 100 -routines 100 -count 1000000

Latency Average: 1.446173ms
Throughput: 533341 /second
```

Now we have **533K/Sec** POST operations using 8 connections. Although latency
is pretty high, 1.4ms, on a local-loop we could attain half-million POST
request using 100 concurrent routines. Concurrency can definitely help
mitigate the high-latency/low-throughput situation, provided server logic
can be as concurrent as the number of clients. And with this experiment, all
8 threads in my macbook-pro are fully saturated.

One last experiment with POST, let us decrease the payload from 512-bytes
(default) to 64-bytes.

```bash
time ./perf -c -addr localhost:9900 -do post -conns 8 -batchsize 100 -routines 100 -payload 64 -count 1000000

Latency Average: 1.089072ms
Throughput: 666674 /second
```

Decreasing the payload improves the latency by 50% and it reflects in
throughput as well.

**REQUEST Benchmark**

Now that we have exercised most of the options, like number-of-connections,
payload-size, batchsize, number-of-routines. For the remaining experiment
we shall stick to 64-byte payload, 100 routines, 100 batchsize.

For singe connection, let us do 1-million request-response.

```bash
time ./perf -c -addr localhost:9900 -do request -conns 1 -batchsize 100 -routines 100 -payload 64 -count 1000000

Latency Average: 2.058103ms
Throughput: 50005
```

For a similar configuration, POST alone can do about 166K/sec. This is because
REQUEST involves rounding trip of 64-byte payload from client to server and
back from server to client, which leads to 2ms latency and just 50K/Sec. throughput.

To decrease the latency reduce batchsize to 1:

```bash
time ./perf -c -addr localhost:9900 -do request -conns 1 -batchsize 1 -routines 1 -payload 64 -count 1000000

Latency Average: 85.735µs
Throughput: 11627
```

We could bring down latency to **85us**, by throughput suffers lets us tune it
further to get better throughput, increase batchsize to 8 and increase
concurrency to 20:

```bash
time ./perf -c -addr localhost:9900 -do request -conns 1 -batchsize 8 -routines 20 -payload 64 -count 1000000

Latency Average: 464.587µs
Throughput: 33334
```

By increasing the number of connections to 8, with batchsize and concurrency
at 100:

```bash
time ./perf -c -addr localhost:9900 -do request -conns 8 -batchsize 100 -routines 100 -payload 64 -count 1000000

Latency Average: 12.843589ms
Throughput: 61544
```

We get good throughput but poor latency.

**STREAM Benchmark**

Client can initiate a stream request, using `-do streamtx` and send `-stream` number of messages to
remote. To initiate a stream request in reverse direction, use `-do streamrx`.

```
time ./perf -c -addr localhost:9900 -do streamtx -batchsize 1 -routines 1 -payload 64 -conns 1 -count 1 -stream 100000

Latency Average: 1.701267364s
Throughput: 100001
```

This run took 1.7 seconds to complete a single stream request and to stream 100K
messages on that request. If we increase the number of stream-request to 2.

```bash
time ./perf -c -addr localhost:9900 -do streamtx -batchsize 1 -routines 1 -payload 64 -conns 1 -count 2 -stream 100000

Latency Average: 1.628144653s
Throughput: 66667
```

Let us increase the routines to 2:

```bash
time ./perf -c -addr localhost:9900 -do streamtx -batchsize 1 -routines 2 -payload 64 -conns 1 -count 2 -stream 100000

Latency Average: 3.596201162s
Throughput: 66667
```

Latency has jumped, this means there is congestion on the socket due to
concurrent routines. Now, let us optimize by increasing batchsize, along with
count and routines

```
time ./perf -c -addr localhost:9900 -do streamtx -batchsize 20 -routines 20 -payload 64 -conns 1 -count 20 -stream 100000

Latency Average: 12.026882441s
Throughput: 166668
```

We have managed to stream messages at 166K/sec rate. But due to increased
congestion on the socket, latency is suffering here. Now let us make lot of
stream request, with each request sending just 10 messages, and increase the
number of stream-requests to 100K:

```bash
time ./perf -c -addr localhost:9900 -do streamtx -batchsize 100 -routines 100 -payload 64 -conns 1 -count 100000  -stream 10

Latency Average: 11.354511ms
Throughput: 91666
```

**100K/Sec. througput** at **11ms** latency. Repeat this on many
connections.

```bash
time ./perf -c -addr localhost:9900 -do streamtx -batchsize 100 -routines 100 -payload 64 -conns 8 -count 100000  -stream 10

Latency Average: 28.065024ms
Throughput: 237837
```

**237K/Sec. througput** at **23ms** latency.
