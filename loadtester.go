package loadtester

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// LoadTester is a load testing tool.
type LoadTester struct {
	httpClient *HttpClient
}

const (
	// DefaultRedirects is the default number of times an Attacker follows
	// redirects.
	DefaultRedirects = 10
	// DefaultTimeout is the default amount of time an Attacker waits for a request
	// before it times out.
	DefaultTimeout = 30 * time.Second
	// DefaultConnections is the default amount of max open idle connections per
	// target host.
	DefaultConnections = 10000
	// DefaultMaxConnections is the default amount of connections per target
	// host.
	DefaultMaxConnections = 0
	// NoFollow is the value when redirects are not followed but marked successful
	NoFollow = -1
)

var (
	// DefaultLocalAddr is the default local IP address an Attacker uses.
	DefaultLocalAddr = net.IPAddr{IP: net.IPv4zero}
	// DefaultTLSConfig is the default tls.Config an Attacker uses.
	DefaultTLSConfig = &tls.Config{InsecureSkipVerify: false}
)

type fn func(ctx context.Context, httpClient *HttpClient)

type Rate struct {
	Freq int
	Per  time.Duration
}

func NewLoadTester() LoadTester {
	l := LoadTester{}

	l.httpClient = NewHTTPClient()

	return l
}

// Start starts the load test.
// ctx is needed for graceful cancellation.
// fn is a function that sends requests to the target. The http client will be passed by LoadTester to the function.
func (l *LoadTester) Start(ctx context.Context, rate Rate, duration time.Duration, fn fn) {
	var wg sync.WaitGroup

	log.Info().Msgf("Starting load test with %d requests per %s for %fs", rate.Freq, rate.Per, duration.Seconds())

	ctxWithCancel, cancel := context.WithCancel(ctx)
	go func() {
		for {
			select {
			case <-ctxWithCancel.Done():
				return
			default:
				for i := 0; i < rate.Freq; i++ {
					wg.Add(1)
					go func() {
						fn(ctx, l.httpClient)
						wg.Done()
					}()
				}
				time.Sleep(rate.Per)
			}
		}
	}()

	select {
	case <-ctx.Done():
	case <-time.After(duration):
	}
	log.Info().Msg("Stopping load test, waiting for all requests to finish...")
	cancel()
	wg.Wait()
	log.Info().Msg("Load test finished!")
}

// SummarizeMetrics returns map[name]Metrics from HttpClient.
func (l *LoadTester) SummarizeMetrics() map[string]Metrics {
	return l.httpClient.SummarizeMetrics()
}

func (l *LoadTester) TablePrintMetrics() {
	l.httpClient.TablePrintMetrics()
}
