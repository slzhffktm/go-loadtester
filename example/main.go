package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/slzhffktm/go-loadtester"
)

func init() {
	// Set up logging.
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func main() {
	loadTester := loadtester.NewLoadTester()

	duration := 10 * time.Second                        // Duration: 10 seconds
	rate := loadtester.Rate{Freq: 10, Per: time.Second} // 10 requests per second

	loadTester.Start(context.Background(), rate, duration, func(ctx context.Context, httpClient *loadtester.HttpClient) {
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

		err = httpClient.SendRequest(
			ctx,
			"Get object",
			http.MethodGet,
			fmt.Sprintf("https://api.restful-api.dev/objects/%v", res["id"]),
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
