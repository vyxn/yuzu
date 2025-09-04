package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

// .config/meta-yuzu/providers/*.json discover all jsons in there
// TODO on setup create folder structure + copy the ones in project for the time being
// TODO xml response support
// TODO include json path library to access results
// TODO return compiled output value

type Provider struct {
	Id        string            `json:"id"`
	Inputs    map[string]string `json:"inputs"`
	Envs      map[string]string `json:"envs"`
	BaseURL   string            `json:"baseUrl"`
	Headers   map[string]string `json:"headers"`
	Endpoints []Endpoint        `json:"endpoints"`
	Output    map[string]string `json:"output"`
	Schema    string            `json:"schema"`
}

type Endpoint struct {
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	URLParams    map[string]string `json:"urlParams,omitempty"`
	Headers      map[string]string `json:"headers"`
	BodyParams   []string          `json:"bodyParams,omitempty"`
	Cache        bool              `json:"cache,omitempty"`
	ResponseType string            `json:"responseType,omitempty"`
	Result       map[string]string `json:"result,omitempty"`
}

var Providers = make(map[string]*Provider)

func (p *Provider) Run(inputs map[string]string) {
	// Merging os.env and inputs for this run environment values
	runEnv := make(map[string]string)
	for k, v := range p.Envs {
		runEnv[k] = v
	}
	for k, v := range p.Inputs {
		runEnv[v] = inputs[k]
	}

	ctx := context.Background()
	client := &http.Client{Timeout: 10 * time.Second}
	for i, e := range p.Endpoints {
		path := getFromRunEnv(runEnv, e.Path)
		u, err := url.Parse(fmt.Sprintf("%s%s", p.BaseURL, path))
		if err != nil {
			println(yerr.WithStackf("parsing url <%s>: %w", p.BaseURL, err).Error())
		}
		slog.Info("formed url", slog.String("url", u.String()))

		q := u.Query()
		for k, v := range e.URLParams {
			q.Add(k, getFromRunEnv(runEnv, v))
		}
		u.RawQuery = q.Encode()

		r, err := http.NewRequestWithContext(ctx, e.Method, u.String(), nil)
		if err != nil {
			println(yerr.WithStackf("creating request <%s>: %w", u.String(), err))
		}

		for k, v := range p.Headers {
			r.Header.Add(k, getFromRunEnv(runEnv, v))
		}
		for k, v := range e.Headers {
			r.Header.Add(k, getFromRunEnv(runEnv, v))
		}

		resp, err := client.Do(r)
		if err != nil {
			println(yerr.WithStackf("fetching <%s>: %w", u.String(), err))
		}
		defer resp.Body.Close()

		if resp.StatusCode < http.StatusOK ||
			resp.StatusCode >= http.StatusMultipleChoices {
			b, _ := io.ReadAll(resp.Body)
			println(yerr.WithStackf("bad status <%s>: %s", resp.Status, string(b)))
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			println(yerr.WithStackf("reading response body: %w", err))
		}

		println(string(body))
		var result map[string]any
		err = json.Unmarshal(body, &result)
		if err != nil {
			panic(err)
		}

		for k := range e.Result {
			if i == 0 {
				runEnv[k] = fmt.Sprintf("%v", result["data"].([]any)[0].(map[string]any)["node"].(map[string]any)["id"].(float64))
			}
			if i == 1 {
				runEnv[k] = result[k].(string)
			}
		}
		println(fmt.Sprintf("%+v", runEnv))
	}
}

func getFromRunEnv(values map[string]string, compilable string) string {
	compiled := compilable
	for k, v := range values {
		compiled = strings.ReplaceAll(compiled, k, v)
	}

	return compiled
}
