package gofast

import "strings"
import "net/http"
import "time"
import _ "fmt"

import "github.com/prataprc/gson"

// Statshandler http handler to handle statistics endpoint, returns
// statistics for specified transport or aggregate statistics of all
// transports, based on the query parameters.
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
			stats = Stats()
		} else {
			stats = Stat(name[0])
		}
		stats = filterstats(stats, keys)
		stats["timestamp"] = uint64(time.Now().UnixNano())

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
