package provider

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"github.com/maypok86/otter/v2"
	"github.com/vyxn/yuzu/internal/pkg/yerr"

	"github.com/AsaiYusuke/jsonpath/v2"
)

type Endpoint struct {
	Method       string            `json:"method"                 jsonschema:"required,enum=GET,POST,PUT,PATCH,DELETE"`
	URL          string            `json:"url"                    jsonschema:"required,format=uri"`
	Params       map[string]string `json:"params,omitempty"       jsonschema:""`
	Headers      map[string]string `json:"headers,omitempty"      jsonschema:""`
	Body         []string          `json:"body,omitempty"         jsonschema:""`
	Cache        bool              `json:"cache,omitempty"        jsonschema:""`
	ResponseType string            `json:"responseType,omitempty" jsonschema:"enum=json,xml,text,binary"`
	Result       map[string]string `json:"result,omitempty"       jsonschema:""`
	provider     *HTTPProvider     `json:"-"                      jsonschema:"-"`
	cache        sync.Map
}

func (e *Endpoint) run(ctx context.Context, runEnv *RunEnv) error {
	u, err := url.Parse(runEnv.Replace(e.URL))
	if err != nil {
		return yerr.WithStackf("parsing url %q: %w", e.URL, err)
	}

	q := u.Query()
	for k, v := range e.Params {
		q.Add(k, runEnv.Replace(v))
	}
	u.RawQuery = q.Encode()

	r, err := http.NewRequestWithContext(ctx, e.Method, u.String(), nil)
	if err != nil {
		return yerr.WithStackf("creating request %q: %w", u.String(), err)
	}

	for k, v := range e.provider.Headers {
		r.Header.Add(k, runEnv.Replace(v))
	}
	for k, v := range e.Headers {
		r.Header.Add(k, runEnv.Replace(v))
	}

	var data []byte
	if e.Cache && r.Method == http.MethodGet {
		data, err = e.provider.Cache.Get(
			ctx,
			r.URL.String(),
			otter.LoaderFunc[string, []byte](
				func(ctx context.Context, key string) ([]byte, error) {
					return e.doRequest(r)
				},
			),
		)
	} else {
		data, err = e.doRequest(r)
	}
	if err != nil {
		return err
	}

	var result any
	err = json.Unmarshal(data, &result)
	if err != nil {
		return yerr.WithStackf(
			"unmarshalling endpoint %q response: %w\n%s",
			u.String(), err, string(data),
		)
	}

	for k, v := range e.Result {
		out, err := jsonpath.Retrieve(runEnv.Replace(v), result)
		if err != nil {
			slog.Info("result value", slog.Any("result", result))
			return yerr.WithStackf("retrieving jsonpath: %w", err)
		}

		runEnv.AddPublic(k, out[0])
	}

	return nil
}

func (e *Endpoint) doRequest(req *http.Request) ([]byte, error) {
	resp, err := e.provider.client.Do(req)
	if err != nil {
		return nil, yerr.WithStackf(
			"fetching %+v %q: %w",
			resp, req.URL.String(), err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		b, _ := io.ReadAll(resp.Body)
		return nil, yerr.WithStackf("bad status %q: %s", resp.Status, string(b))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, yerr.WithStackf("reading response body: %w", err)
	}

	return body, nil
}
