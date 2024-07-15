package loadtester

import (
	"time"

	"github.com/influxdata/tdigest"
)

type Metrics struct {
	// Latencies holds computed request latency metrics.
	Latencies LatencyMetrics `json:"latencies"`
	// LatenciesSuccess holds computed request latency metrics for the requests that successfully receive responses only.
	LatenciesSuccess LatencyMetrics `json:"latencies_success"`
	// Requests is the total number of requests executed.
	Requests uint64 `json:"requests"`
	// Success is the percentage of non-error responses.
	Success float64 `json:"success"`
	// StatusCodes is a histogram of the responses' status codes.
	StatusCodes map[int]int `json:"status_codes"`
	// Errors is a set of unique errors returned by the targets during the attack.
	Errors []string `json:"errors"`

	errors       map[string]struct{}
	SuccessCount uint64
}

func newMetrics() Metrics {
	return Metrics{
		Latencies:        newLatencyMetrics(),
		LatenciesSuccess: newLatencyMetrics(),
		StatusCodes:      map[int]int{},
		errors:           map[string]struct{}{},
		Errors:           make([]string, 0),
	}
}

type LatencyMetrics struct {
	// Total is the total latency sum of all requests in an attack.
	Total time.Duration `json:"total"`
	// Mean is the mean request latency.
	Mean time.Duration `json:"mean"`
	// P50 is the 50th percentile request latency.
	P50 time.Duration `json:"50th"`
	// P90 is the 90th percentile request latency.
	P90 time.Duration `json:"90th"`
	// P95 is the 95th percentile request latency.
	P95 time.Duration `json:"95th"`
	// P99 is the 99th percentile request latency.
	P99 time.Duration `json:"99th"`
	// Max is the maximum observed request latency.
	Max time.Duration `json:"max"`
	// Min is the minimum observed request latency.
	Min time.Duration `json:"min"`

	estimator estimator
}

// Add adds the given latency to the latency metrics.
func (l *LatencyMetrics) Add(latency time.Duration) {
	if l.Total += latency; latency > l.Max {
		l.Max = latency
	}
	if latency < l.Min || l.Min == 0 {
		l.Min = latency
	}
	l.estimator.Add(float64(latency))
}

// Quantile returns the nth quantile from the latency summary.
func (l *LatencyMetrics) Quantile(nth float64) time.Duration {
	return time.Duration(l.estimator.Get(nth))
}

func newLatencyMetrics() LatencyMetrics {
	return LatencyMetrics{
		estimator: newTdigestEstimator(100),
	}
}

type estimator interface {
	Add(sample float64)
	Get(quantile float64) float64
}

type tdigestEstimator struct{ *tdigest.TDigest }

func newTdigestEstimator(compression float64) *tdigestEstimator {
	return &tdigestEstimator{TDigest: tdigest.NewWithCompression(compression)}
}

func (e *tdigestEstimator) Add(s float64) {
	e.TDigest.Add(s, 1)
}

func (e *tdigestEstimator) Get(q float64) float64 {
	return e.TDigest.Quantile(q)
}
