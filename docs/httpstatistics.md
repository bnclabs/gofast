# HTTP endpoints

Out of the box Gofast support http endpoints to gather transport statistics.
It can be very useful for plotting the live graphs in dashboards and/or for
off site logging and network monitoring.

To enable http endpoints applications need to import the `http/` sub-package
and endpoints are automatically mounted using net/http.DefaultServerMux.

```go
import _ github.com/bnclabs/gofast/http
```

Subsequently it is expected that application is going to spawn a http server
using the default multiplexer:

```
go func() {
    log.Println(http.ListenAndServe(":8080", nil))
}()
```

Following endpoints are automatically mounted:

```text
/gofast/transports
/gofast/statistics?name=<transport-name>
/gofast/statistics?keys=n_tx,n_rx
/gofast/memstats
```

If `name` query-parameter is supplied, complete set of statistics for
the specified <transport-name> will be returned as JSON text.

If `name` query-parameter is skipped, endpoint returns complete set of
statistics for all transport objects.

If `keys` query-parameter is supplied, as comma separated values of
count-name, only specified list of count values will be returned as JSON
text.

Note that `name` and `keys` parameter can be mixed.

Other than supported stats keys, http response object will also include
`timestamp` at which statistic was gathered.

## Example access

Gather list of active transports.

```bash
curl http://localhost:8080/gofast/transports

["server-0","server-1","server-10","server-11","server-12","server-13",
 "server-14","server-15","server-2","server-3","server-4","server-5",
"server-6","server-7","server-8","server-9"]
```

Gather statistics for active transports.

```bash
$ curl http://localhost:8080/gofast/statistics

{ "n_dropped":0,    "n_flushes":1824,      "n_mdrops":0,  "n_rx":59249033,
  "n_rxbeats":1756, "n_rxbyte":5747043869, "n_rxfin":0,   "n_rxpost":59249001,
  "n_rxreq":16,     "n_rxresp":16,         "n_rxstart":0, "n_rxstream":0,
  "n_tx":1824,      "n_txbyte":55180,      "n_txfin":0,   "n_txpost":1792,
  "n_txreq":16,     "n_txresp":16,         "n_txstart":0, "n_txstream":0,
  "timestamp":1518173082250829000
}
```

Gather statistics for transport by name `server-1`

```bash
$ curl http://localhost:8080/gofast/statistics\?name\=server-1

{ "n_dropped":0,  "n_flushes":50,       "n_mdrops":0,  "n_rx":1669159,
  "n_rxbeats":47, "n_rxbyte":161905264, "n_rxfin":0,   "n_rxpost":1669157,
  "n_rxreq":1,    "n_rxresp":1,         "n_rxstart":0, "n_rxstream":0,
  "n_tx":50,      "n_txbyte":1528,      "n_txfin":0,   "n_txpost":48,
  "n_txreq":1,    "n_txresp":1,         "n_txstart":0,  "n_txstream":0,
  "timestamp":1518173017791050000
}
```

Gather statistics `n_rxpost` and `n_txpost` for transport `server-1`

```bash
$ curl http://localhost:8080/gofast/statistics\?name\=server-1\&keys\=n_rxpost,n_txpost

{"n_rxpost":3436102,"n_txpost":105,"timestamp":1518173074484234000}
```

Gather memory GC statistics, note that this applies to the entire program.

``` bash
$ curl http://localhost:8080/gofast/memstats

{ "frees":6069769, "gcsys":15024128, "heapalloc":288667736, "heapidle":80691200,
  "heapinuse":290275328, "heapreleased":0, "heapsys":370966528,
  "mallocs":7381461, "numgc":10, "pausens":, "pausetotalns":1172650
}
```
