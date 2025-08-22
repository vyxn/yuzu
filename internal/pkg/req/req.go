// Package req includes utilities to simplify http calls
package req

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/vyxn/yuzu/internal/pkg/log"
)

var logger *slog.Logger

func init() {
	logger = log.NewLogger()
}

const timeout = 10 * time.Second

var client = &http.Client{Timeout: timeout}

func Get(
	ctx context.Context,
	url string,
	headers map[string]string,
) ([]byte, error) {
	logger.Info(
		"→ r",
		slog.String("method", http.MethodGet),
		slog.String("url", url),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error bad status <%s>: %s", resp.Status, string(b))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}
