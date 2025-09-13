package provider

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

type ProviderFinder interface {
	Get(id string) (Provider, error)
}

type Provider interface {
	ID() string
	ClearID()
	MimeType() string
	Run(context.Context, map[string]string) ([]byte, error)
}

type RawProvider struct {
	Type string `json:"type"`
	Raw  json.RawMessage
}

func (p *RawProvider) UnmarshalJSON(data []byte) error {
	p.Raw = append(p.Raw[:0], data...)

	type Alias RawProvider
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	return json.Unmarshal(data, &aux)
}

func NewProvider(path string, r io.Reader) (Provider, error) {
	var p RawProvider
	d := json.NewDecoder(r)
	if err := d.Decode(&p); err != nil {
		return nil, yerr.WithStackf("unmarshaling provider JSON: %w", err)
	}

	switch p.Type {
	case "http":
		return newHTTPProvider(path, &p)
	case "cli":
		slog.Warn("cli provider not yet implemented")
		return nil, nil
	default:
		slog.Warn("provider type not supported", slog.String("type", p.Type))
		return nil, nil
	}
}
