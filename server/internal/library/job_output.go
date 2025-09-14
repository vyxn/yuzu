package library

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"

	"github.com/vyxn/yuzu/internal/output"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
	"github.com/vyxn/yuzu/internal/pkg/zip"
	"github.com/vyxn/yuzu/internal/provider"
)

type JobOutput interface {
	Run(runEnv *provider.RunEnv, data any) error
}

type RawJobOutput struct {
	ID      string `json:"id"`
	Raw     json.RawMessage
	library *Library `json:"-"`
}

func (jo *RawJobOutput) UnmarshalJSON(data []byte) error {
	jo.Raw = append(jo.Raw[:0], data...)

	type Alias RawJobOutput
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(jo),
	}

	return json.Unmarshal(data, &aux)
}

func (jo *RawJobOutput) Resolve() (JobOutput, error) {
	out, err := jo.library.outputFinder.Get(jo.ID)
	if err != nil {
		return nil, err
	}

	switch out.GetType() {
	case "file":
		return newJobFileOutput([]byte(jo.Raw), out)
	default:
		return nil, yerr.WithStackf("invalid output type")
	}
}

type JobFileOutput struct {
	ID      string        `json:"id"`
	Path    string        `json:"path"`
	ZipPath string        `json:"zipPath"`
	out     output.Output `json:"-"`
}

func newJobFileOutput(data []byte, out output.Output) (*JobFileOutput, error) {
	var jfo JobFileOutput
	if err := json.Unmarshal(data, &jfo); err != nil {
		return nil, err
	}
	jfo.out = out
	return &jfo, nil
}

func (o *JobFileOutput) Run(runEnv *provider.RunEnv, data any) (ferr error) {
	path := runEnv.Replace(o.Path)
	if o.ZipPath != "" {
		zipPath := runEnv.Replace(o.ZipPath)
		return o.writeInsideZip(zipPath, path, data)
	} else {
		return o.onlyFileWrite(path, data)
	}
}

func (o *JobFileOutput) onlyFileWrite(path string, data any) (ferr error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			ferr = errors.Join(ferr, err)
			return
		}
	}()

	return o.out.Run(data, f)
}

func (o *JobFileOutput) writeInsideZip(
	zipPath string,
	pathInZip string,
	data any,
) (ferr error) {
	pr, pw := io.Pipe()
	go func() {
		defer func() {
			if err := pw.Close(); err != nil {
				ferr = errors.Join(ferr, err)
				return
			}
		}()

		err := o.out.Run(data, pw)
		if err != nil {
			slog.Error("writing in zip", slog.Any("error", err))
		}
	}()

	return zip.WriteInZip(zipPath, pathInZip, pr)
}
