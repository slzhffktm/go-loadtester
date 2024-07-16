package loadtester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
)

// HttpClient is the wrapper of net/http Client that has the ability to collect metrics.
type HttpClient struct {
	dialer *net.Dialer
	client http.Client
	stats  []stat
	// mu guards stats.
	mu sync.Mutex
}

type stat struct {
	// name is defined by the user and is used to identify the stats for different URLs (for example).
	name       string
	statusCode int
	latency    time.Duration
	err        string
}

func NewHTTPClient() *HttpClient {
	dialer := &net.Dialer{
		LocalAddr: &net.TCPAddr{IP: DefaultLocalAddr.IP, Zone: DefaultLocalAddr.Zone},
		KeepAlive: 30 * time.Second,
	}

	return &HttpClient{
		dialer: dialer,
		client: http.Client{
			Timeout: DefaultTimeout,
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				DialContext:         dialer.DialContext,
				TLSClientConfig:     DefaultTLSConfig,
				MaxIdleConnsPerHost: DefaultConnections,
				MaxConnsPerHost:     DefaultMaxConnections,
			},
		},
		stats: []stat{},
	}
}

func (h *HttpClient) SendRequest(
	ctx context.Context,
	name string,
	method string,
	requestUrl string,
	headers map[string]string,
	request any,
	queryParameters url.Values,
	response any,
) error {
	var body []byte
	var err error

	reqUrl, err := url.Parse(requestUrl)
	if err != nil {
		return fmt.Errorf("url.Parse: %w", err)
	}

	if request != nil {
		body, err = json.Marshal(request)
		if err != nil {
			return fmt.Errorf("json.Marshal request body: %w", err)
		}
	}

	if queryParameters != nil {
		reqUrl.RawQuery = queryParameters.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqUrl.String(), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("http.NewRequestWithContext: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	startTime := time.Now()
	httpRes, err := h.client.Do(req)
	if err != nil {
		h.addStat(name, 0, time.Since(startTime), err)
		return fmt.Errorf("client.Do: %w", err)
	}

	resBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return err
	}
	httpRes.Body.Close()

	// We always expect 2xx, otherwise mark as error.
	if httpRes.StatusCode >= 200 && httpRes.StatusCode < 300 {
		h.addStat(name, httpRes.StatusCode, time.Since(startTime), nil)
	} else {
		h.addStat(
			name,
			httpRes.StatusCode,
			time.Since(startTime),
			fmt.Errorf("unexpected status code %d, response body: %s", httpRes.StatusCode, string(resBody)),
		)
		return fmt.Errorf("unexpected status code %d, response body: %s", httpRes.StatusCode, string(resBody))
	}

	if err := json.Unmarshal(resBody, response); err != nil {
		return fmt.Errorf("json.Unmarshal response body: %w, response body: %s", err, string(resBody))
	}

	return nil
}

func (h *HttpClient) addStat(name string, statusCode int, latency time.Duration, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	errString := ""
	if err != nil {
		errString = err.Error()
	}
	h.stats = append(h.stats, stat{
		name:       name,
		statusCode: statusCode,
		latency:    latency,
		err:        errString,
	})
}

// SummarizeMetrics returns map[name]Metrics.
func (h *HttpClient) SummarizeMetrics() map[string]Metrics {
	mapMetrics := map[string]Metrics{}

	for _, s := range h.stats {
		metrics, ok := mapMetrics[s.name]
		if !ok {
			metrics = newMetrics()
			mapMetrics[s.name] = metrics
		}
		metrics.Requests++
		metrics.StatusCodes[s.statusCode]++
		metrics.Latencies.Add(s.latency)
		if s.statusCode >= 200 && s.statusCode < 300 {
			metrics.LatenciesSuccess.Add(s.latency)
			metrics.SuccessCount++
		}

		if s.err != "" {
			if _, ok := metrics.errors[s.err]; !ok {
				metrics.errors[s.err] = struct{}{}
				metrics.Errors = append(metrics.Errors, s.err)
			}
		}

		mapMetrics[s.name] = metrics
	}

	for k, metrics := range mapMetrics {
		metrics.Success = float64(metrics.SuccessCount) / float64(metrics.Requests)
		metrics.Latencies.Mean = time.Duration(float64(metrics.Latencies.Total) / float64(metrics.Requests))
		metrics.Latencies.P50 = metrics.Latencies.Quantile(0.50)
		metrics.Latencies.P90 = metrics.Latencies.Quantile(0.90)
		metrics.Latencies.P95 = metrics.Latencies.Quantile(0.95)
		metrics.Latencies.P99 = metrics.Latencies.Quantile(0.99)

		metrics.LatenciesSuccess.Mean = time.Duration(float64(metrics.LatenciesSuccess.Total) / float64(metrics.SuccessCount))
		metrics.LatenciesSuccess.P50 = metrics.LatenciesSuccess.Quantile(0.50)
		metrics.LatenciesSuccess.P90 = metrics.LatenciesSuccess.Quantile(0.90)
		metrics.LatenciesSuccess.P95 = metrics.LatenciesSuccess.Quantile(0.95)
		metrics.LatenciesSuccess.P99 = metrics.LatenciesSuccess.Quantile(0.99)
		mapMetrics[k] = metrics
	}

	return mapMetrics
}

func (h *HttpClient) TablePrintMetrics() {
	metricsMap := h.SummarizeMetrics()

	h.tablePrintSummaries(metricsMap)
	h.tablePrintStatusCodes(metricsMap)
	h.tablePrintLatencies(metricsMap)
	h.tablePrintLatenciesSuccess(metricsMap)
}

func (h *HttpClient) tablePrintSummaries(metricsMap map[string]Metrics) {
	fmt.Println()
	fmt.Println("Summaries")
	summaries := tablewriter.NewWriter(os.Stdout)
	summaries.SetRowSeparator("-")
	summaries.SetHeader([]string{
		"Name",
		"Total Requests",
		"Success %",
		"Total Errors",
		"Error Lists",
	})
	summaries.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgRedColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgRedColor},
	)
	for k, metrics := range metricsMap {
		summaries.Append([]string{
			k,
			fmt.Sprintf("%d", metrics.Requests),
			fmt.Sprintf("%.2f %%", metrics.Success*100),
			fmt.Sprintf("%d", metrics.Requests-metrics.SuccessCount),
			fmt.Sprintf("%s", metrics.Errors),
		})
	}
	summaries.Render()
	fmt.Println()
}

func (*HttpClient) tablePrintStatusCodes(metricsMap map[string]Metrics) {
	fmt.Println("Status Codes")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetRowSeparator("-")

	// Retrieve unique status codes. )
	statusCodeSet := map[int]struct{}{}
	for _, metrics := range metricsMap {
		for k := range metrics.StatusCodes {
			statusCodeSet[k] = struct{}{}
		}
	}
	statusCodesUnique := make([]int, 0, len(statusCodeSet))
	for k := range statusCodeSet {
		statusCodesUnique = append(statusCodesUnique, k)
	}
	sort.Ints(statusCodesUnique)
	headers := []string{
		"Name",
	}
	headerColors := []tablewriter.Colors{
		{tablewriter.Bold, tablewriter.FgCyanColor},
	}
	for _, v := range statusCodesUnique {
		headers = append(headers, fmt.Sprintf("%d", v))
		if v >= 200 && v < 300 {
			headerColors = append(headerColors, tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor})
		} else {
			headerColors = append(headerColors, tablewriter.Colors{tablewriter.Bold, tablewriter.FgRedColor})
		}
	}
	table.SetHeader(headers)
	table.SetHeaderColor(headerColors...)
	for k, metrics := range metricsMap {
		rows := []string{k}
		for _, v := range statusCodesUnique {
			rows = append(rows, fmt.Sprintf("%d", metrics.StatusCodes[v]))
		}
		table.Append(rows)
	}
	table.Render()
	fmt.Println()
}

func (*HttpClient) tablePrintLatencies(metricsMap map[string]Metrics) {
	fmt.Println("Latencies")
	latencies := tablewriter.NewWriter(os.Stdout)
	latencies.SetRowSeparator("-")
	latencies.SetHeader([]string{
		"Name",
		"50%",
		"90%",
		"95%",
		"99%",
		"Avg",
		"Max",
		"Min",
	})
	latencies.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
	)
	for k, metrics := range metricsMap {
		latencies.Append([]string{
			k,
			fmt.Sprintf("%d ms", metrics.Latencies.P50.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.Latencies.P90.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.Latencies.P95.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.Latencies.P99.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.Latencies.Mean.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.Latencies.Max.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.Latencies.Min.Milliseconds()),
		})
	}
	latencies.Render()
	fmt.Println()
}

func (*HttpClient) tablePrintLatenciesSuccess(metricsMap map[string]Metrics) {
	fmt.Println("Latencies (Success Only)")
	latenciesSuccess := tablewriter.NewWriter(os.Stdout)
	latenciesSuccess.SetRowSeparator("-")
	latenciesSuccess.SetHeader([]string{
		"Name",
		"50%",
		"90%",
		"95%",
		"99%",
		"Avg",
		"Max",
		"Min",
	})
	latenciesSuccess.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
	)
	for k, metrics := range metricsMap {
		latenciesSuccess.Append([]string{
			k,
			fmt.Sprintf("%d ms", metrics.LatenciesSuccess.P50.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.LatenciesSuccess.P90.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.LatenciesSuccess.P95.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.LatenciesSuccess.P99.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.LatenciesSuccess.Mean.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.LatenciesSuccess.Max.Milliseconds()),
			fmt.Sprintf("%d ms", metrics.LatenciesSuccess.Min.Milliseconds()),
		})
	}
	latenciesSuccess.Render()
	fmt.Println()
}
