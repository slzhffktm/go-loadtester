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

	duration := 10 * time.Second                        // Duration: 10 seconds
	rate := loadtester.Rate{Freq: 10, Per: time.Second} // 10 requests per second

	loadTester.Start(context.Background(), rate, duration, func(ctx context.Context, httpClient *loadtester.HttpClient) {
		// Send the first HTTP request.
		var res map[string]any
		err := httpClient.SendRequest(
			ctx,
			"Create object",
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
			"Get object",
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
