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
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
	"github.com/vyxn/yuzu/internal/utils"

	"github.com/AsaiYusuke/jsonpath/v2"
	"github.com/goccy/go-yaml"
	"github.com/kaptinlin/jsonschema"
	xmlparser "github.com/moolekkari/validatexml-go"
)

type HTTPProvider struct {
	ID        string            `json:"-"                 jsonschema:"-"`
	Path      string            `json:"-"                 jsonschema:"-"`
	Type      string            `json:"type"              jsonschema:"required,enum=http"`
	Inputs    map[string]string `json:"inputs"            jsonschema:"required,minProperties=1"`
	Envs      map[string]string `json:"envs,omitempty"    jsonschema:""`
	Vars      map[string]string `json:"vars,omitempty"    jsonschema:""`
	Headers   map[string]string `json:"headers,omitempty" jsonschema:""`
	Endpoints []Endpoint        `json:"endpoints"         jsonschema:"required"`
	Output    Output            `json:"output"            jsonschema:"required"`
}

type Endpoint struct {
	Method       string            `json:"method"                 jsonschema:"required,enum=GET,POST,PUT,PATCH,DELETE"`
	URL          string            `json:"url"                    jsonschema:"required,format=uri"`
	Params       map[string]string `json:"params,omitempty"       jsonschema:""`
	Headers      map[string]string `json:"headers,omitempty"      jsonschema:""`
	Body         []string          `json:"body,omitempty"         jsonschema:""`
	Cache        bool              `json:"cache,omitempty"        jsonschema:""`
	ResponseType string            `json:"responseType,omitempty" jsonschema:"enum=json,xml,text,binary"`
	Result       map[string]string `json:"result,omitempty"       jsonschema:""`
}

type Output struct {
	Type       string             `json:"type"    jsonschema:"required,enum=json,md,xml,yaml"`
	Schema     string             `json:"schema"  jsonschema:""`
	Content    map[string]any     `json:"content" jsonschema:"required"`
	JSONSchema *jsonschema.Schema `json:"-"       jsonschema:"-"`
	XMLSchema  *xmlparser.Schema  `json:"-"       jsonschema:"-"`
}

func newHTTPProvider(path string, rp *RawProvider) (*HTTPProvider, error) {
	schema := jsonschema.FromStruct[HTTPProvider]()
	res := schema.ValidateJSON(rp.Raw)

	if !res.IsValid() {
		errs := ""
		for path, message := range res.GetDetailedErrors() {
			errs += fmt.Sprintf("\n- %s: %s", path, message)
			slog.Info(
				"validation result",
				slog.String("path", path),
				slog.String("message", message),
			)

		}
		return nil, yerr.WithStackf("validating provider json schema: %s", errs)
	}

	var provider HTTPProvider
	if err := json.Unmarshal(rp.Raw, &provider); err != nil {
		return nil, yerr.WithStackf("unmarshaling provider JSON: %v", err)
	}

	provider.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	provider.Path = path

	envs := make(map[string]string)
	for env, placeholder := range provider.Envs {
		if v, ok := os.LookupEnv(env); ok {
			envs[placeholder] = v
		} else {
			slog.Warn(
				"provider env is not configured",
				slog.String("provider", provider.ID),
				slog.String("env", env),
				slog.String("placeholder", placeholder),
			)
		}
	}
	provider.Envs = envs

	if provider.Output.Schema != "" {
		rawSchema, err := utils.GetLocalOrRemoteFile(provider.Output.Schema)
		if err != nil {
			return nil, err
		}

		switch provider.Output.Type {
		case "json", "md", "yaml":
			compiler := jsonschema.NewCompiler()
			schema, err := compiler.Compile(rawSchema)
			if err != nil {
				return nil, yerr.WithStackf("parsing schema: %w", err)
			}

			provider.Output.JSONSchema = schema
		case "xml":
			schema, err := xmlparser.ParseXSD(rawSchema)
			if err != nil {
				return nil, yerr.WithStackf("parsing schema: %w", err)
			}

			provider.Output.XMLSchema = schema
		}
	}

	return &provider, nil
}

func (p *HTTPProvider) ProviderID() string {
	return p.ID
}

func (p *HTTPProvider) GetPath() string {
	return p.Path
}

func (p *HTTPProvider) Store(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(p)
}

func (p *HTTPProvider) MimeType() string {
	switch p.Output.Type {
	case "json":
		return "application/json"
	case "md":
		return "application/md"
	case "xml":
		return "application/xml"
	case "yaml":
		return "application/yaml"
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

	return p.generateOutput(utils.SubstituteKeys(runEnv, p.Output.Content))
}

func (p *HTTPProvider) generateOutput(content any) ([]byte, error) {
	var output []byte
	var err error
	var data []byte
	var dataErr error

	switch p.Output.Type {
	case "json":
		output, err = json.MarshalIndent(content, "", "  ")
	case "md":
		output, err = yaml.MarshalWithOptions(content, yaml.Indent(2))
		output = slices.Concat([]byte("---\n"), output, []byte("\n---\n"))
		data, dataErr = json.Marshal(content)
	case "xml":
		output, err = MapToXML(content)
	case "yaml":
		output, err = yaml.MarshalWithOptions(content, yaml.Indent(2))
		data, dataErr = json.Marshal(content)

	default:
		return nil, yerr.WithStackf("output type %s not supported", p.Output.Type)
	}

	if err != nil {
		return nil, yerr.WithStackf("marshalling to %s: %w", p.Output.Type, err)
	}

	if dataErr != nil {
		return nil, yerr.WithStackf("marshalling data to %s: %w", p.Output.Type, dataErr)
	}

	if data == nil {
		data = output
	}
	if err := p.validateOutputSchema(data); err != nil {
		return nil, err
	}

	return output, nil
}

func (p *HTTPProvider) validateOutputSchema(data []byte) error {
	if p.Output.Schema == "" {
		return nil
	}

	switch p.Output.Type {
	case "json", "md", "yaml":
		res := p.Output.JSONSchema.ValidateJSON(data)

		if !res.IsValid() {
			errs := ""
			for path, message := range res.GetDetailedErrors() {
				errs += fmt.Sprintf("\n- %s: %s", path, message)
				slog.Info(
					"validation result",
					slog.String("path", path),
					slog.String("message", message),
				)

			}
			return yerr.WithStackf("validating json output: %s", errs)
		}

		return nil
	case "xml":
		doc, err := xmlparser.Parse(data)
		if err != nil {
			return yerr.WithStackf("parsing output data: %w", err)
		}

		return p.Output.XMLSchema.Validate(doc)
	default:
		return nil
	}
}

func getFromRunEnv(values map[string]string, compilable string) string {
	compiled := compilable
	for k, v := range values {
		compiled = strings.ReplaceAll(compiled, k, v)
	}

	return compiled
}
