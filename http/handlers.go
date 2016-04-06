// http package to use gofast http endpoints, subscribed
// to net/http.DefaultServerMux.
//
// import _ github.com/prataprc/gofast/http
//
// will automatically mount,
//
//   /gofast/statistics?name=<transport-name>
//   /gofast/statistics?keys=n_tx,n_rx
//
// * if `name` query-parameter is supplied, complete set of statistics
// for the specified <transport-name> will be returned as JSON
// text.
//
// * if `name` query-parameter is skipped, endpoint returns complete set
// of statistics for all transport objects.
//
// * if `keys` query-parameter is supplied, as comma separated values of
// count-name, only specified list of count values will be returned
// as JSON text.
package http

import "strings"
import "net/http"
import "encoding/json"
import "fmt"

import "github.com/prataprc/gofast"

func init() {
	http.HandleFunc("/gofast/statistics", statshandler)
}

func Statshandler(w http.ResponseWriter, r *http.Request) {
	if query := r.URL.Query(); query != nil {
		name, _ := query["name"]
		keyparam, _ := query["keys"]
		keys := []string{}
		if len(keyparam) > 0 {
			keys = strings.Split(strings.Trim(keyparam[0], " \r\n\t"), ",")
		}

		var stats map[string]uint64
		if len(name) == 0 {
			stats = gofast.Stats()
		} else {
			stats = gofast.Stat(name[0])
		}
		fmt.Println(stats)
		stats = filterstats(stats, keys)
		fmt.Println(stats)

		jsonstats, err := json.Marshal(stats)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error() + "\n"))
			return
		}

		header := w.Header()
		header["Content-Type"] = []string{"application/json"}
		w.WriteHeader(200)
		w.Write(jsonstats)
		w.Write([]byte("\n"))
	}
}

func filterstats(stats map[string]uint64, keys []string) map[string]uint64 {
	if len(keys) == 0 {
		return stats
	}
	m := map[string]uint64{}
	for _, key := range keys {
		m[key] = stats[key]
	}
	return m
}
