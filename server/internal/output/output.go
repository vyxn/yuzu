// Package output handles the operation to be performed with the data collected
// from all the configured providers from a library
package output

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

type OutputFinder interface {
	Get(id string) (Output, error)
}

type Output interface {
	GetID() string
	ClearID()
	GetType() string
	Run(data any, w io.Writer) error
}

type RawOutput struct {
	Type string `json:"type"`
	Raw  json.RawMessage
}

func (p *RawOutput) UnmarshalJSON(data []byte) error {
	p.Raw = append(p.Raw[:0], data...)

	type Alias RawOutput
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	return json.Unmarshal(data, &aux)
}

func NewOutput(path string, r io.Reader) (Output, error) {
	var o RawOutput
	d := json.NewDecoder(r)
	if err := d.Decode(&o); err != nil {
		return nil, yerr.WithStackf("unmarshaling provider JSON: %w", err)
	}

	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	switch o.Type {
	case "file":
		return newFileOutput(id, &o)
	default:
		return nil, yerr.WithStackf("output %q not supported", o.Type)
	}
}
