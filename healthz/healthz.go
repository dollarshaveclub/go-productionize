package healthz // import "github.com/dollarshaveclub/go-productionize/healthz"

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/pprof"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/dollarshaveclub/go-productionize/reporter"
	"github.com/dollarshaveclub/go-productionize/svcinfo"
)

const (
	healthy = iota
	unhealthy
)

const (
	rootPrefix = "/healthz"

	indexHTMLTemplate = `
<html>
	<head><title>healthz</title></head>

	<body>
		<h1>Healthz Endpoints</h1>
		<ul>
			<li><a href="{{ .Prefix }}/ready">{{ .Prefix }}/ready</a></li>
			<li><a href="{{ .Prefix }}/alive">{{ .Prefix }}/alive</a></li>
			<li><a href="{{ .Prefix }}/stats">{{ .Prefix }}/stats</a></li>
			<li><a href="{{ .Prefix }}/diediedie">{{ .Prefix }}/diediedie</a> (Kills the app)</li>
			<li><a href="{{ .Prefix }}/abortabortabort">{{ .Prefix }}/abortabortabort</a> (Kills the pod!)</li>
			<li><a href="{{ .Prefix }}/pprof">{{ .Prefix }}/pprof</a></li>
		</ul>
	</body>
</html>`
)

var (
	ready bool

	statuses = []statusInfo{
		{
			HTTPCode:    http.StatusOK,
			Description: "healthy",
		},
		{
			HTTPCode:    http.StatusServiceUnavailable,
			Description: "unhealthy",
		},
	}
)

type statusInfo struct {
	HTTPCode    int    `json:"http_code"`
	Description string `json:"description"`
}

// Healthz struct
type Healthz struct {
	alive  int
	ready  int
	Status int

	dd     *statsd.Client
	ddRate float64

	reporter *reporter.Reporter

	sync.RWMutex
}

// New will create a new healthz object and setup a new HTTP server mux
func New(mux *http.ServeMux, opts ...func(*Healthz)) (*Healthz, error) {
	if mux == nil {
		return nil, errors.New("you must provide a http.ServeMux")
	}

	h := &Healthz{
		alive:  unhealthy,
		ready:  unhealthy,
		Status: unhealthy,
	}

	for _, opt := range opts {
		opt(h)
	}

	// path.Join didn't work for the indexHandler here so...
	mux.HandleFunc(rootPrefix+"/", indexHandler())
	mux.HandleFunc(path.Join(rootPrefix, "ready"), h.readyHandler())
	mux.HandleFunc(path.Join(rootPrefix, "alive"), h.livelinessHandler())
	mux.HandleFunc(path.Join(rootPrefix, "stats"), h.statsHandler())

	mux.HandleFunc(path.Join(rootPrefix, "diediedie"), h.dieDieDieHandler())
	mux.HandleFunc(path.Join(rootPrefix, "abortabortabort"), h.abortAbortAbortHandler())

	// Add pprof outside of the default HTTP ServeMux
	//
	// path.Join doesn't work here and causes bad path to be created on the resulting
	// index page from the net/http/pprof package
	mux.HandleFunc(rootPrefix+"/pprof/", fakePProfIndexHandler())
	mux.HandleFunc(path.Join(rootPrefix, "pprof/cmdline"), pprof.Cmdline)
	mux.HandleFunc(path.Join(rootPrefix, "pprof/profile"), pprof.Profile)
	mux.HandleFunc(path.Join(rootPrefix, "pprof/symbol"), pprof.Symbol)
	mux.HandleFunc(path.Join(rootPrefix, "pprof/trace"), pprof.Trace)

	return h, nil
}

// SetDataDogClient adds the DD client to the healthz endpoints for tracking
func SetDataDogClient(dd *statsd.Client, rate float64) func(*Healthz) {
	return func(h *Healthz) {
		h.dd = dd
		h.ddRate = rate
	}
}

// SetReporter adds an instance of the reporter to get stats about the running app
func SetReporter(r *reporter.Reporter) func(*Healthz) {
	return func(h *Healthz) {
		h.reporter = r
	}
}

// SetAlive sets the service to live as soon as the health port is started
func SetAlive() func(*Healthz) {
	return func(h *Healthz) {
		h.alive = healthy
	}
}

// indexHandler will provide the user with a basic HTML page that provies access to the available
// endpoints provided by this library
func indexHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := template.New("index").Parse(indexHTMLTemplate)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "parsing the HTML template failed: %+v", err)
		}

		info := struct {
			Prefix string
		}{
			Prefix: rootPrefix,
		}

		err = t.Execute(w, info)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error writting index template: %+v", err)
		}
	}
}

// fakePProfIndex will rewrite the path that is sent to the server as the pprof.Index function is
// hardcoded to expect it to be under "/debug/pprof". So here we strip off the rootPrefix for the
// healthz endpoint and replace it with "/debug" since we keep the rest under the expected paths
func fakePProfIndexHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = path.Join("/debug", strings.TrimPrefix(r.URL.Path, rootPrefix))
		pprof.Index(w, r)
	}
}

// Ready will set the readiness endpoint to a good value
func (h *Healthz) Ready() {
	h.Lock()
	h.ready = healthy
	h.Unlock()
	return
}

// NotReady will set the readiness endpoint to report the service as unhealthy
func (h *Healthz) NotReady() {
	h.Lock()
	h.ready = unhealthy
	h.Unlock()
	return
}

// ReadyHandler will be used to handle requests to /healthz/ready
func (h *Healthz) readyHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		h.RLock()
		ready := h.ready
		h.RUnlock()

		if h.dd != nil {
			defer h.dd.TimeInMilliseconds("healthz.ready", float64(time.Since(start)*time.Millisecond), []string{fmt.Sprintf("ready:%t", ready == healthy)}, h.ddRate)
		}

		w.WriteHeader(statuses[ready].HTTPCode)
		fmt.Fprintf(w, statuses[ready].Description)
	}
}

// Alive will set the liveliness endpoint to a good value
func (h *Healthz) Alive() {
	h.Lock()
	h.alive = healthy
	h.Unlock()
	return
}

// NotAlive will set the liveliness endpoint to report the service as unhealthy
func (h *Healthz) NotAlive() {
	h.Lock()
	h.alive = unhealthy
	h.Unlock()
	return
}

// LivelinessHandler will be used to handle requests to /healthz/ready
func (h *Healthz) livelinessHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		h.RLock()
		alive := h.alive
		h.RUnlock()

		if h.dd != nil {
			h.dd.TimeInMilliseconds("healthz.alive", float64(time.Since(start)*time.Millisecond), []string{fmt.Sprintf("alive:%t", alive == healthy)}, h.ddRate)
		}

		w.WriteHeader(statuses[alive].HTTPCode)
		fmt.Fprintf(w, statuses[alive].Description)
	}
}

func (h *Healthz) statsHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		start := time.Now()

		info := struct {
			Status    statusInfo            `json:"status"`
			SvcInfo   svcinfo.ServiceInfo   `json:"service_info"`
			GoInfo    reporter.GoInfo       `json:"go_info"`
			GoRuntime reporter.RuntimeStats `json:"runtime"`
		}{
			Status: statuses[h.Status],
		}

		if h.reporter != nil {
			stats := h.reporter.GetStats()
			info.GoInfo = stats.GoInfo
			info.GoRuntime = stats.Runtime
			info.SvcInfo = stats.ServiceInfo
		}

		if h.dd != nil {
			defer h.dd.TimeInMilliseconds("healthz.stats", float64(time.Since(start)*time.Millisecond), []string{fmt.Sprintf("error:%t", err != nil)}, h.ddRate)
		}

		infoJSON, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error getting stats: %v", err)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(infoJSON)
	}
}

// dieDieDieHandler will quickly terminate the server
func (h *Healthz) dieDieDieHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		os.Exit(255)
	}
}

// abortAbortAbortHandler will set the ready and alive endpoints to report unhealthy so that
// k8s will take care of it
func (h *Healthz) abortAbortAbortHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		h.NotReady()
		h.NotAlive()

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "App reporting as unhealthy...")
	}
}
