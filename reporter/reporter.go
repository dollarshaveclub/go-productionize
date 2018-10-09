package reporter // import "github.com/dollarshaveclub/go-productionize/reporter"

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/dollarshaveclub/go-productionize/svcinfo"
)

const (
	// DefaultPeriod is the amount of time between when we send out updated metrics
	DefaultPeriod = 5 * time.Second
)

var (
	defaultTags = []string{}
)

// Reporter provides basic statistics about the runtime
type Reporter struct {
	ctx        context.Context
	cancelFunc func()
	dd         *statsd.Client
	period     time.Duration
	stats      Stats

	defaultTags []string
	infoTags    []string
	runtimeTags []string

	sync.RWMutex
}

// GoInfo provides information about the Go Toolchain used to build the binary
type GoInfo struct {
	Arch    string
	OS      string
	Version string
}

// RuntimeStats contains runtime information about Go
type RuntimeStats struct {
	MaxProcs     int
	NumCPU       int
	NumCgoCall   int64
	NumGoroutine int
}

// Stats struct keeps track of all information
type Stats struct {
	GoInfo      GoInfo
	Mem         runtime.MemStats
	Runtime     RuntimeStats
	ServiceInfo svcinfo.ServiceInfo
}

// New will create a new runtime watch background process and begin producing metrics for the runtime
//
// A new Reporter is returned to allow the service to manually pull a set of stats if required.
func New(dd *statsd.Client, options ...func(*Reporter)) *Reporter {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Reporter{
		ctx:         ctx,
		cancelFunc:  cancel,
		dd:          dd,
		period:      DefaultPeriod,
		stats:       Stats{},
		defaultTags: defaultTags,
	}

	for _, o := range options {
		o(r)
	}

	// These are static so we only need to store these once
	r.stats.ServiceInfo = svcinfo.GetInfo()

	r.stats.GoInfo = GoInfo{
		Version: runtime.Version(),
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
	}

	r.stats.Mem = runtime.MemStats{}
	r.stats.Runtime = RuntimeStats{}
	r.buildTags()

	r.dd.Count("startup", 1, []string{}, 1.0)

	r.watch()
	return r
}

// Period can be used as an option function for the New function above. The period is the amount
// of time between stat lookup and push to DataDog. This should be used as an option to the New
// function above.
func Period(t time.Duration) func(*Reporter) {
	return func(r *Reporter) {
		r.period = t
	}
}

// DefaultTags allows you to add a set of tags to every metric sent. This should be used as an option
// to the New function above.
func DefaultTags(tags []string) func(*Reporter) {
	return func(r *Reporter) {
		r.defaultTags = tags
	}
}

func (r *Reporter) watch() {
	go func() {
		var start time.Time
		var processTime time.Duration
		lastGCNum := uint32(0)
		timer := time.NewTimer(0)
		for {
			select {
			case <-timer.C:
				start = time.Now()

				// Store this so that we can record all of the GCs that occurred since last lookup
				lastGCNum = r.stats.Mem.NumGC

				r.fillStats()

				r.exportMemStats(lastGCNum)
				r.exportRuntimeStats()
				r.exportRuntimeInfo()
				r.exportServiceInfo()

				processTime = time.Since(start)

				// Try to produce stats every 5 seconds so remove the processing time from the sleep
				if r.period-processTime > 0 {
					timer = time.NewTimer(r.period - processTime)
				}
			case <-r.ctx.Done():
				return
			}
		}
	}()
}

func (r *Reporter) buildTags() {
	// Skip over the default tags so that we don't double tag if these were added before
	r.infoTags = svcinfo.GetDDTags()

	// Tags are used here to allow for these values to be graphed as needed
	r.runtimeTags = r.defaultTags
	r.runtimeTags = append(r.runtimeTags, fmt.Sprintf("version:%s", r.stats.GoInfo.Version), fmt.Sprintf("arch:%s", r.stats.GoInfo.Arch), fmt.Sprintf("os:%s", r.stats.GoInfo.OS))
}

func (r *Reporter) fillStats() {
	// Don't want to read while we're updating pointers
	r.Lock()

	runtime.ReadMemStats(&r.stats.Mem)

	r.stats.Runtime = RuntimeStats{
		NumCPU:       runtime.NumCPU(),
		NumCgoCall:   runtime.NumCgoCall(),
		NumGoroutine: runtime.NumGoroutine(),
		MaxProcs:     runtime.GOMAXPROCS(0),
	}

	r.Unlock()
}

func (r *Reporter) exportRuntimeStats() {
	r.dd.Gauge("go.num_cpu", float64(r.stats.Runtime.NumCPU), r.defaultTags, 1.0)
	r.dd.Gauge("go.num_cgo_calls", float64(r.stats.Runtime.NumCgoCall), r.defaultTags, 1.0)
	r.dd.Gauge("go.goroutines", float64(r.stats.Runtime.NumGoroutine), r.defaultTags, 1.0)
	r.dd.Gauge("go.max_procs", float64(r.stats.Runtime.MaxProcs), r.defaultTags, 1.0)
}

func (r *Reporter) exportRuntimeInfo() {
	r.dd.Gauge("go.runtime", 1.0, r.runtimeTags, 1.0)
}

func (r *Reporter) exportServiceInfo() {
	// Send some basic service information constantly to track for the duration of the services' operation
	if len(r.infoTags) > 0 {
		r.dd.Gauge("info", 1.0, r.infoTags, 1.0)
	}
}

func (r *Reporter) exportMemStats(lastGCNum uint32) {
	r.dd.Gauge("go.mem.total_alloc", float64(r.stats.Mem.TotalAlloc), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.alloc", float64(r.stats.Mem.Alloc), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.sys", float64(r.stats.Mem.Sys), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.lookups", float64(r.stats.Mem.Lookups), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.mallocs", float64(r.stats.Mem.Mallocs), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.frees", float64(r.stats.Mem.Frees), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.heap_alloc", float64(r.stats.Mem.HeapAlloc), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.heap_sys", float64(r.stats.Mem.HeapSys), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.heap_idle", float64(r.stats.Mem.HeapIdle), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.heap_inuse", float64(r.stats.Mem.HeapInuse), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.heap_released", float64(r.stats.Mem.HeapReleased), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.heap_objects", float64(r.stats.Mem.HeapObjects), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.stack_inuse", float64(r.stats.Mem.StackInuse), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.stack_sys", float64(r.stats.Mem.StackSys), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.mspan_inuse", float64(r.stats.Mem.MSpanInuse), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.mspan_sys", float64(r.stats.Mem.MSpanSys), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.mcache_inuse", float64(r.stats.Mem.MCacheInuse), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.mcache_sys", float64(r.stats.Mem.MCacheSys), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.buck_hash_sys", float64(r.stats.Mem.BuckHashSys), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.gc_sys", float64(r.stats.Mem.GCSys), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.other_sys", float64(r.stats.Mem.OtherSys), r.defaultTags, 1.0)

	r.dd.Gauge("go.mem.next_gc", float64(r.stats.Mem.NextGC), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.last_gc", float64(r.stats.Mem.LastGC), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.pause_total_ns", float64(r.stats.Mem.PauseTotalNs), r.defaultTags, 1.0)

	// Ring around the ring buffer
	for i := lastGCNum; i <= r.stats.Mem.NumGC; i++ {
		t, err := time.ParseDuration(fmt.Sprintf("%dns", r.stats.Mem.PauseNs[i%256]))
		if err != nil {
			continue
		}
		r.dd.Timing("go.mem.pause_ns", t, r.defaultTags, 1.0)
	}

	r.dd.Gauge("go.mem.num_gc", float64(r.stats.Mem.NumGC), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.num_forced_gc", float64(r.stats.Mem.NumForcedGC), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.gc_cpu_fraction", r.stats.Mem.GCCPUFraction, r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.num_forced_gc", float64(r.stats.Mem.NumForcedGC), r.defaultTags, 1.0)
	r.dd.Gauge("go.mem.num_forced_gc", float64(r.stats.Mem.NumForcedGC), r.defaultTags, 1.0)

	var tags []string
	for _, b := range r.stats.Mem.BySize {
		tags = r.defaultTags
		tags = append(tags, fmt.Sprintf("size:%d", b.Size))

		r.dd.Gauge("go.mem.mallocs_by_size", float64(b.Mallocs), tags, 1.0)
		r.dd.Gauge("go.mem.frees_by_size", float64(b.Frees), tags, 1.0)
	}
}

// GetStats will return a set of stats to the requester
func (r *Reporter) GetStats() Stats {
	r.RLock()
	defer r.RUnlock()

	return r.stats
}

// Cancel will stop the watcher
func (r *Reporter) Cancel() {
	r.cancelFunc()
}
