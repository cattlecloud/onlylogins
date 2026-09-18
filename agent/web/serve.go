package web

import (
	"net/http"
	"time"

	"cattlecloud.net/go/scope"
	"cattlecloud.net/go/ulog"
	"github.com/VictoriaMetrics/metrics"
)

// timeout is the nominal amount of time to allow clients to complete requests
const timeout = 10 * time.Second

// methods represents the mapping between supported HTTP methods and the
// implementation thereof.
type methods map[string]methodfuncs

// methodfuncs is an expanded version of ServeHTTP that accepts an additional
// context and forces an error return.
type methodfuncs func(scope.C, http.ResponseWriter, *http.Request) error

// serve retrives the appropriate handler function from methods and calls it,
// using the error value to print an error log if necessary.
func serve(
	ctx scope.C,
	log *ulog.Log,
	w http.ResponseWriter,
	r *http.Request,
	methods map[string]methodfuncs,
) {
	f, exists := methods[r.Method]
	if !exists {
		log.E.Fmt("no implementation for method %s", r.Method)
		return
	}

	if err := f(ctx, w, r); err != nil {
		log.E.Fmt("catastrophic web handler failure: %v", err)
		return
	}
}

// duration is a middleware for tracking request duration; assigning the value
// to the metric associated with the request path
type duration struct {
	sum  *metrics.Summary
	next http.Handler
}

func (d *duration) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// start the clock for tracking page load time
	start := time.Now()

	// execute the page
	d.next.ServeHTTP(w, r)

	// only care about GET requests; page load performance
	if r.Method == http.MethodGet {
		d.sum.UpdateDuration(start)
	}
}
