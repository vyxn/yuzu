package provider

import (
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const defaultTimeout = 10 * time.Second

type APIClient struct {
	client     *http.Client
	limiter    *rate.Limiter
	maxRetries int
	cooldown   time.Duration
}

func NewAPIClient(
	limiter *rate.Limiter,
	maxRetries int,
	cooldown time.Duration,
) *APIClient {
	return &APIClient{
		client:     &http.Client{Timeout: defaultTimeout},
		limiter:    limiter,
		maxRetries: maxRetries,
		cooldown:   cooldown,
	}
}

func (c *APIClient) Do(req *http.Request) (*http.Response, error) {
	var err error
	for attempt := range c.maxRetries {
		if err := c.limiter.Wait(req.Context()); err != nil {
			return nil, err
		}

		var resp *http.Response
		resp, err = c.client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, err
		}

		slog.Warn(
			"request retry",
			slog.Int("attempt", attempt),
			slog.Any("error", err),
		)

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < c.maxRetries {
			time.Sleep(c.cooldown)
		}
	}

	return nil, err
}
