// Package req includes utilities to simplify http calls
package req

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

const timeout = 10 * time.Second

var client = &http.Client{Timeout: timeout}

func Get(
	ctx context.Context,
	url string,
	headers map[string]string,
) ([]byte, error) {
	slog.Info(
		"→ r",
		slog.String("method", http.MethodGet),
		slog.String("url", url),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, yerr.WithStackf("creating request <%s>: %w", url, err)
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, yerr.WithStackf("fetching <%s>: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		b, _ := io.ReadAll(resp.Body)
		return nil, yerr.WithStackf("bad status <%s>: %s", resp.Status, string(b))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, yerr.WithStackf("reading response body: %w", err)
	}

	return body, nil
}
