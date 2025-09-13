package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
	"github.com/vyxn/yuzu/internal/utils"

	"github.com/goccy/go-yaml"
	"github.com/kaptinlin/jsonschema"
	"github.com/maypok86/otter/v2"
	"github.com/maypok86/otter/v2/stats"
	xmlparser "github.com/moolekkari/validatexml-go"
	"golang.org/x/time/rate"
)

const defaultBurst = 1
const defaultMaxRetry = 5
const defaultCooldown = 2 * time.Second

var defaultRateLimit = rate.Every(time.Second)

type HTTPProvider struct {
	Id        string                       `json:"id,omitempty"      jsonschema:"-"`
	Type      string                       `json:"type"              jsonschema:"required,enum=http"`
	HTTP      *HTTP                        `json:"http,omitempty"    jsonschema:""`
	Inputs    map[string]string            `json:"inputs"            jsonschema:"required,minProperties=1"`
	Envs      map[string]string            `json:"envs,omitempty"    jsonschema:""`
	Vars      map[string]string            `json:"vars,omitempty"    jsonschema:""`
	Headers   map[string]string            `json:"headers,omitempty" jsonschema:""`
	Endpoints []*Endpoint                  `json:"endpoints"         jsonschema:"required"`
	Output    Output                       `json:"output"            jsonschema:"required"`
	client    *APIClient                   `json:"-"                 jsonschema:"-"`
	Cache     *otter.Cache[string, []byte] `json:"-"                 jsonschema:"-"`
}

type Output struct {
	Type       string             `json:"type"    jsonschema:"required,enum=json,md,xml,yaml"`
	Schema     string             `json:"schema"  jsonschema:""`
	Content    map[string]any     `json:"content" jsonschema:"required"`
	JSONSchema *jsonschema.Schema `json:"-"       jsonschema:"-"`
	XMLSchema  *xmlparser.Schema  `json:"-"       jsonschema:"-"`
}

type HTTP struct {
	Timeout           time.Duration `json:"timeout,omitempty"           jsonschema:""`
	RequestsPerSecond float64       `json:"requestsPerSecond,omitempty" jsonschema:""`
	Burst             int           `json:"burst,omitempty"             jsonschema:""`
	MaxRetries        int           `json:"maxRetries,omitempty"        jsonschema:""`
	Cooldown          time.Duration `json:"cooldown,omitempty"          jsonschema:""`
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
		return nil, yerr.WithStackf(
			"validating provider json schema: %s",
			errs,
		)
	}

	var provider HTTPProvider

	// TODO: would be better to unmarshall with the schema, to use schema default
	// values, but for now this is not working, review after a while.
	// if err := schema.Unmarshal(&provider, rp.Raw); err != nil {
	if err := json.Unmarshal(rp.Raw, &provider); err != nil {
		return nil, yerr.WithStackf("unmarshaling provider JSON: %v", err)
	}

	provider.Id = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	for _, e := range provider.Endpoints {
		e.provider = &provider
	}

	envs := make(map[string]string)
	for env, placeholder := range provider.Envs {
		if v, ok := os.LookupEnv(env); ok {
			envs[placeholder] = v
		} else {
			slog.Warn(
				"provider env is not configured",
				slog.String("provider", provider.Id),
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

	rateLimit := defaultRateLimit
	burst := defaultBurst
	maxRetry := defaultMaxRetry
	cooldown := defaultCooldown
	if provider.HTTP != nil {
		if provider.HTTP.RequestsPerSecond != 0 {
			rateLimit = rate.Limit(provider.HTTP.RequestsPerSecond)
		}
		if provider.HTTP.Burst != 0 {
			burst = provider.HTTP.Burst
		}

		if provider.HTTP.MaxRetries != 0 {
			maxRetry = provider.HTTP.MaxRetries
		}
		if provider.HTTP.Cooldown != 0 {
			cooldown = provider.HTTP.Cooldown
		}
	}
	limiter := rate.NewLimiter(rateLimit, burst)

	cache, err := otter.New(&otter.Options[string, []byte]{
		MaximumSize:      10_000,
		InitialCapacity:  1_000,
		ExpiryCalculator: otter.ExpiryWriting[string, []byte](10 * time.Minute),
		StatsRecorder:    stats.NewCounter(),
	})
	assert.Assert(err == nil, "invalid cache config")

	provider.Cache = cache
	provider.client = NewAPIClient(limiter, maxRetry, cooldown)

	return &provider, nil
}

func (p *HTTPProvider) ID() string {
	return p.Id
}

func (p *HTTPProvider) ClearID() {
	p.Id = ""
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

func (p *HTTPProvider) Run(
	ctx context.Context,
	inputs map[string]string,
) ([]byte, error) {
	runEnv := p.newRunEnv(inputs)

	for _, e := range p.Endpoints {
		if err := e.run(ctx, runEnv); err != nil {
			slog.Error("running endpoint", slog.Any("error", err))
		}
	}

	return p.generateOutput(runEnv.ReplaceAny(p.Output.Content))
}

func (p *HTTPProvider) newRunEnv(inputs map[string]string) *RunEnv {
	public := make(map[string]string)
	maps.Copy(public, p.Vars)
	for k, v := range p.Inputs {
		public[v] = inputs[k]
	}
	return NewRunEnv(public, p.Envs)
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
		return nil, yerr.WithStackf(
			"output type %s not supported",
			p.Output.Type,
		)
	}

	if err != nil {
		return nil, yerr.WithStackf(
			"marshalling to %s: %w",
			p.Output.Type,
			err,
		)
	}

	if dataErr != nil {
		return nil, yerr.WithStackf(
			"marshalling data to %s: %w",
			p.Output.Type,
			dataErr,
		)
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
