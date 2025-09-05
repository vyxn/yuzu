package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AsaiYusuke/jsonpath/v2"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

// .config/meta-yuzu/providers/*.json discover all jsons in there
// TODO on setup create folder structure + copy the ones in project for the time being
// TODO xml response support
// TODO include json path library to access results
// TODO return compiled output value

type Provider struct {
	ID        string            `json:"id"`
	Inputs    map[string]string `json:"inputs"`
	Envs      map[string]string `json:"envs"`
	BaseURL   string            `json:"baseUrl"`
	Headers   map[string]string `json:"headers"`
	Endpoints []Endpoint        `json:"endpoints"`
	Output    Output            `json:"output"`
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

type Output struct {
	Type    string            `json:"type"`
	Schema  string            `json:"schema"`
	Content map[string]string `json:"content"`
}

var Providers = make(map[string]*Provider)

func (p *Provider) MimeType() string {
	switch p.Output.Type {
	case "json":
		return "application/json"
	case "xml":
		return "application/xml"
	default:
		return ""
	}
}

func (p *Provider) Run(inputs map[string]string) ([]byte, error) {
	// Merging os.env and inputs for this run environment values
	runEnv := make(map[string]string)
	maps.Copy(runEnv, p.Envs)
	for k, v := range p.Inputs {
		runEnv[v] = inputs[k]
	}

	slog.Info("runEnv", slog.Any("", runEnv))

	ctx := context.Background()
	client := &http.Client{Timeout: 10 * time.Second}
	for _, e := range p.Endpoints {
		path := getFromRunEnv(runEnv, e.Path)
		u, err := url.Parse(fmt.Sprintf("%s%s", p.BaseURL, path))
		if err != nil {
			return nil, yerr.WithStackf("parsing url <%s>: %w", p.BaseURL, err)
		}
		slog.Info("formed url", slog.String("url", u.String()))

		q := u.Query()
		for k, v := range e.URLParams {
			q.Add(k, getFromRunEnv(runEnv, v))
		}
		u.RawQuery = q.Encode()

		r, err := http.NewRequestWithContext(ctx, e.Method, u.String(), nil)
		if err != nil {
			return nil, yerr.WithStackf("creating request <%s>: %w", u.String(), err)
		}

		for k, v := range p.Headers {
			r.Header.Add(k, getFromRunEnv(runEnv, v))
		}
		for k, v := range e.Headers {
			r.Header.Add(k, getFromRunEnv(runEnv, v))
		}

		resp, err := client.Do(r)
		if err != nil {
			return nil, yerr.WithStackf("fetching <%s>: %w", u.String(), err)
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

		var result any
		err = json.Unmarshal(body, &result)
		if err != nil {
			return nil, yerr.WithStackf(
				"unmarshalling endpoint <%s> response: %w\n%s",
				u.String(),
				err,
				string(body),
			)
		}

		for k, v := range e.Result {
			out, err := jsonpath.Retrieve(v, result)
			if err != nil {
				return nil, yerr.WithStackf("retrieving jsonpath: %w", err)
			}

			switch v := out[0].(type) {
			case string:
				runEnv[k] = v
			case float64:
				runEnv[k] = strconv.FormatFloat(v, 'f', -1, 64)
			case bool:
				if v {
					runEnv[k] = "true"
				} else {
					runEnv[k] = "false"
				}
			default:
				return nil, yerr.WithStackf("retrieved value has unsupported type <%v>: %w", v, err)
			}

		}
		// println(fmt.Sprintf("%+v", runEnv))
	}

	output := map[string]any{}
	for k, v := range p.Output.Content {
		output[k] = getFromRunEnv(runEnv, v)
	}

	// Generate Output
	switch p.Output.Type {
	case "json":
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return nil, yerr.WithStackf("marshalling to json: %w", err)
		}

		return data, nil
	case "xml":
		data, err := MapToXML(output, "content")
		if err != nil {
			return nil, yerr.WithStackf("marshalling to xml: %w", err)
		}

		return data, nil

	default:
		return nil, yerr.WithStackf("output type %s not supported", p.Output.Type)
	}
}

func getFromRunEnv(values map[string]string, compilable string) string {
	compiled := compilable
	for k, v := range values {
		compiled = strings.ReplaceAll(compiled, k, v)
	}

	return compiled
}
