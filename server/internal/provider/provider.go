package provider

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vyxn/yuzu/internal/pkg/yerr"

	"github.com/AsaiYusuke/jsonpath/v2"
)

type Provider interface {
	Run(map[string]string) ([]byte, error)
}

type RawProvider struct {
	Type string `json:"type"`
	Raw  json.RawMessage
}

func New(path string, r io.Reader) (Provider, error) {
	var p RawProvider
	d := json.NewDecoder(r)
	if err := d.Decode(&p); err != nil {
		return nil, yerr.WithStackf("unmarshaling provider JSON: %v", err)
	}

	var prov Provider
	var err error
	switch p.Type {
	case "http":
		prov, err = NewProvider(path)
	case "cli":
	default:
		slog.Warn("provider type not supported", slog.String("type", p.Type))
	}
	return prov, err
}

type HTTPProvider struct {
	ID        string            `json:"id"`
	Inputs    map[string]string `json:"inputs"`
	Envs      map[string]string `json:"envs,omitempty"`
	Vars      map[string]string `json:"vars,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Endpoints []Endpoint        `json:"endpoints"`
	Output    Output            `json:"output"`
	Schema    string            `json:"schema"`
}

type Endpoint struct {
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Params       map[string]string `json:"params,omitempty"`
	Headers      map[string]string `json:"headers"`
	Body         []string          `json:"body,omitempty"`
	Cache        bool              `json:"cache,omitempty"`
	ResponseType string            `json:"responseType,omitempty"`
	Result       map[string]string `json:"result,omitempty"`
}

type Output struct {
	Type    string            `json:"type"`
	Schema  string            `json:"schema"`
	Content map[string]string `json:"content"`
}

func NewProvider(filePath string) (*HTTPProvider, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, yerr.WithStackf("opening provider file: %v", err)
	}

	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, yerr.WithStackf("reading provider file: %v", err)
	}

	var provider HTTPProvider
	if err := json.Unmarshal(bytes, &provider); err != nil {
		return nil, yerr.WithStackf("unmarshaling provider JSON: %v", err)
	}

	m := make(map[string]string)
	for env, placeholder := range provider.Envs {
		if v, ok := os.LookupEnv(env); ok {
			m[placeholder] = v
		} else {
			slog.Warn(
				"provider env is not configured",
				slog.String("provider", provider.ID),
				slog.String("env", env),
				slog.String("placeholder", placeholder),
			)
		}
	}
	provider.Envs = m

	return &provider, nil
}

func (p *HTTPProvider) MimeType() string {
	switch p.Output.Type {
	case "json":
		return "application/json"
	case "xml":
		return "application/xml"
	default:
		return ""
	}
}

func (p *HTTPProvider) Run(inputs map[string]string) ([]byte, error) {
	// Merging os.env and inputs for this run environment values
	runEnv := make(map[string]string)
	maps.Copy(runEnv, p.Envs)
	maps.Copy(runEnv, p.Vars)
	for k, v := range p.Inputs {
		runEnv[v] = inputs[k]
	}

	slog.Info("runEnv", slog.Any("", runEnv))

	ctx := context.Background()
	client := &http.Client{Timeout: 10 * time.Second}
	for _, e := range p.Endpoints {
		u, err := url.Parse(getFromRunEnv(runEnv, e.URL))
		if err != nil {
			return nil, yerr.WithStackf("parsing url <%s> -> <%s>: %w", e.URL, u, err)
		}
		slog.Info("calling endpoint",
			slog.String("url", u.String()),
			slog.Any("runEnv", runEnv),
		)

		q := u.Query()
		for k, v := range e.Params {
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
			out, err := jsonpath.Retrieve(getFromRunEnv(runEnv, v), result)
			if err != nil {
				slog.Info("result value", slog.Any("result", result))
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
			// TODO perhaps this does not make sense if we move to map[string]any
			case nil:
				runEnv[k] = ""
			default:
				return nil, yerr.WithStackf("retrieved value has unsupported type <%v>: %w", v, err)
			}

		}
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
