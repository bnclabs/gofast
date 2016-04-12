// http package to handle gofast http endpoints, subscribed
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

import "net/http"

import "github.com/prataprc/gofast"

func init() {
	http.HandleFunc("/gofast/statistics", gofast.Statshandler)
}
