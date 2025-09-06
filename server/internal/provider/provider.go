package provider

import (
	"encoding/json"
	"io"
	"log/slog"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

type Provider interface {
	ProviderID() string
	Store(io.Writer) error
	Run(map[string]string) ([]byte, error)
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

func New(id string, r io.Reader) (Provider, error) {
	var p RawProvider
	d := json.NewDecoder(r)
	if err := d.Decode(&p); err != nil {
		return nil, yerr.WithStackf("unmarshaling provider JSON: %v", err)
	}

	var prov Provider
	var err error
	switch p.Type {
	case "http":
		prov, err = newHTTPProvider(id, &p)
	case "cli":
	default:
		slog.Warn("provider type not supported", slog.String("type", p.Type))
	}
	return prov, err
}
