# go-loadtester

Simple load testing tool for Go. It is designed to support multistep HTTP testing and will store & calculate the HTTP metrics.

## Why did I build this tool?

Because the existing tools cannot support getting the HTTP response, do something, then use it for the next request.
Example: Create an object, then get the created object using the ID returned from the create object response.
This tool is specifically designed for this purpose.

## Installation

```bash
go get -u github.com/slzhffktm/go-loadtester
```

## How to Use

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/slzhffktm/go-loadtester"
)

func main() {
	loadTester := loadtester.NewLoadTester()

	// Duration: 10 seconds
	duration := 10 * time.Second
	// The rate here means 10 function calls per second (not 10 http calls per second).
	rate := loadtester.Rate{Freq: 10, Per: time.Second}

	loadTester.Start(context.Background(), rate, duration, func(ctx context.Context, httpClient *loadtester.HttpClient) {
		// Send the first HTTP request.
		var res map[string]any
		err := httpClient.SendRequest(
			ctx,
			"Create object", // The name here is used to group the metrics.
			http.MethodPost,
			"https://api.restful-api.dev/objects",
			map[string]string{
				"Content-Type": "application/json",
			},
			map[string]any{
				"name": "Some object" + uuid.NewString(),
				"data": map[string]any{
					"year":           2019,
					"price":          1849.99,
					"CPU model":      "Intel Core i9",
					"Hard disk size": "1 TB",
				},
			},
			nil,
			&res,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create object")
			return
		}

		// Do something with the response from the 1st request, e.g. get the object id.
		id, ok := res["id"].(string)
		if !ok {
			log.Error().Msg("Failed to get object id")
			return
		}

		// Send the 2nd request, using the id from the 1st response.
		err = httpClient.SendRequest(
			ctx,
			"Get object", // Since this request has different name from the 1st one, the metrics will be separated.
			http.MethodGet,
			fmt.Sprintf("https://api.restful-api.dev/objects/%s", id),
			nil,
			nil,
			nil,
			&res,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get object")
		}
	})

	loadTester.TablePrintMetrics()
}
```

It'll print the metrics of each group of the http calls:

```bash
Summaries
+---------------+----------------+-----------+--------------+-------------+
|     NAME      | TOTAL REQUESTS | SUCCESS % | TOTAL ERRORS | ERROR LISTS |
+---------------+----------------+-----------+--------------+-------------+
| Create object |            100 | 100.00 %  |            0 | []          |
| Get object    |            100 | 100.00 %  |            0 | []          |
+---------------+----------------+-----------+--------------+-------------+

Status Codes
+---------------+-----+
|     NAME      | 200 |
+---------------+-----+
| Create object | 100 |
| Get object    | 100 |
+---------------+-----+

Latencies
+---------------+--------+--------+--------+--------+--------+--------+-------+
|     NAME      |  50%   |  90%   |  95%   |  99%   |  AVG   |  MAX   |  MIN  |
+---------------+--------+--------+--------+--------+--------+--------+-------+
| Create object | 106 ms | 233 ms | 348 ms | 356 ms | 129 ms | 357 ms | 88 ms |
| Get object    | 86 ms  | 96 ms  | 99 ms  | 106 ms | 87 ms  | 107 ms | 75 ms |
+---------------+--------+--------+--------+--------+--------+--------+-------+

Latencies (Success Only)
+---------------+--------+--------+--------+--------+--------+--------+-------+
|     NAME      |  50%   |  90%   |  95%   |  99%   |  AVG   |  MAX   |  MIN  |
+---------------+--------+--------+--------+--------+--------+--------+-------+
| Create object | 106 ms | 233 ms | 348 ms | 356 ms | 129 ms | 357 ms | 88 ms |
| Get object    | 86 ms  | 96 ms  | 99 ms  | 106 ms | 87 ms  | 107 ms | 75 ms |
+---------------+--------+--------+--------+--------+--------+--------+-------+
```

## Thanks to

- [vegeta](https://github.com/tsenart/vegeta/tree/master)
- [autocannon-go](https://github.com/GlenTiki/autocannon-go/tree/master)
