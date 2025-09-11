package provider

import (
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const defaultTimeout = 10 * time.Second

type APIClient struct {
	client     *http.Client
	limiter    *rate.Limiter
	maxRetries uint
	cooldown   time.Duration
}

func NewAPIClient(
	limiter *rate.Limiter,
	maxRetries uint,
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
	// var err error
	// for attempt := range c.maxRetries {
	if err := c.limiter.Wait(req.Context()); err != nil {
		return nil, err
	}

	return c.client.Do(req)
	// 	var resp *http.Response
	// 	resp, err = c.client.Do(req)
	// 	if err == nil && resp.StatusCode < 500 {
	// 		// success or client-side error (don’t retry)
	// 		return resp, err
	// 	}
	//
	// 	resp.Body.Close()
	//
	// 	if attempt < c.maxRetries {
	// 		time.Sleep(c.cooldown)
	// 	}
	// }
	//
	// return c.client.Do(req)
}
