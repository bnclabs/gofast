// Package http implement gofast http endpoints, subscribed
// to net/http.DefaultServerMux.
//
// import _ github.com/prataprc/gofast/http
//
// will automatically mount,
//
//   /gofast/statistics?name=<transport-name>
//   /gofast/statistics?keys=n_tx,n_rx
//   /gofast/memstats
//
//   * If `name` query-parameter is supplied, complete set of statistics
//     for the specified <transport-name> will be returned as JSON text.
//   * If `name` query-parameter is skipped, endpoint returns complete set
//     of statistics for all transport objects.
//   * If `keys` query-parameter is supplied, as comma separated values of
//     count-name, only specified list of count values will be returned
//     as JSON text.
//	 * Note that `name` and `keys` parameter can be mixed.
//   * Other than supported stats keys, http response object will also include
//     `timestamp` at which statistic was gathered.
package http

import "runtime"
import "strings"
import "net/http"

import "github.com/prataprc/gson"
import "github.com/prataprc/gofast"

func init() {
	http.HandleFunc("/gofast/statistics", gofast.Statshandler)
	http.HandleFunc("/gofast/memstats", memstats)
}

var fmemsg = strings.Replace(`memstats {
"Alloc":%v, "TotalAlloc":%v, "Sys":%v, "Lookups":%v, "Mallocs":%v,
"Frees":%v, "HeapAlloc":%v, "HeapSys":%v, "HeapIdle":%v, "HeapInuse":%v,
"HeapReleased":%v, "HeapObjects":%v,
"GCSys":%v, "LastGC":%v,
"PauseTotalNs":%v, "PauseNs":%v, "NumGC":%v
}`, "\n", "", -1)

var oldNumGC uint32

func memstats(w http.ResponseWriter, r *http.Request) {
	var ms runtime.MemStats
	var pauseNs [256]uint64

	runtime.ReadMemStats(&ms)
	pausens := newPauseNs(pauseNs[:], ms.PauseNs[:], oldNumGC, ms.NumGC)
	stats := map[string]interface{}{
		// alloc graph
		"mallocs": ms.Mallocs,
		"frees":   ms.Frees,
		// mem graph
		"heapsys":      ms.HeapSys,
		"heapalloc":    ms.HeapAlloc,
		"heapidle":     ms.HeapIdle,
		"heapinuse":    ms.HeapInuse,
		"heapreleased": ms.HeapReleased,
		// pause graph
		"gcsys":        ms.GCSys,
		"pausetotalns": ms.PauseTotalNs,
		"pausens":      pausens,
		"numgc":        ms.NumGC,
	}
	oldNumGC = ms.NumGC

	// marshal and send
	buf, conf := make([]byte, 1024), gson.NewDefaultConfig()
	jsonstats := conf.NewValue(stats).Tojson(conf.NewJson(buf, 0)).Bytes()

	// TODO: remove this once gson becomes stable.
	//jsonstats, err := json.Marshal(stats)
	//if err != nil {
	//	w.WriteHeader(http.StatusInternalServerError)
	//	w.Write([]byte(err.Error() + "\n"))
	//	return
	//}

	header := w.Header()
	header["Content-Type"] = []string{"application/json"}
	header["Access-Control-Allow-Origin"] = []string{"*"}
	w.WriteHeader(200)
	w.Write(jsonstats)
	w.Write([]byte("\n"))
}

func newPauseNs(pad, pauseNs []uint64, oldcount, newcount uint32) []uint64 {
	diff := (newcount - oldcount)
	if diff >= 256 {
		return pauseNs[:]
	}
	for i, j := 0, oldcount+1; j <= newcount; i, j = i+1, j+1 {
		pad[i] = pauseNs[(j+255)%256]
	}
	return pad[:diff]
}
