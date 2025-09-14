package output

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"slices"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
	"github.com/vyxn/yuzu/internal/utils"

	"github.com/goccy/go-yaml"
	"github.com/kaptinlin/jsonschema"
	xmlparser "github.com/moolekkari/validatexml-go"
)

type FileOutput struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Format     string             `json:"format"`
	Schema     string             `json:"schema"`
	jsonSchema *jsonschema.Schema `json:"-"      jsonschema:"-"`
	xmlSchema  *xmlparser.Schema  `json:"-"      jsonschema:"-"`
}

func newFileOutput(id string, ro *RawOutput) (*FileOutput, error) {
	var out FileOutput
	if err := json.Unmarshal(ro.Raw, &out); err != nil {
		return nil, yerr.WithStackf("unmarshaling output %q: %w", id, err)
	}

	out.ID = id

	if out.Schema != "" {
		rawSchema, err := utils.GetLocalOrRemoteFile(out.Schema)
		if err != nil {
			return nil, err
		}

		switch out.Format {
		case "json", "md", "yaml":
			compiler := jsonschema.NewCompiler()
			schema, err := compiler.Compile(rawSchema)
			if err != nil {
				return nil, yerr.WithStackf("parsing schema: %w", err)
			}

			out.jsonSchema = schema
		case "xml":
			schema, err := xmlparser.ParseXSD(rawSchema)
			if err != nil {
				return nil, yerr.WithStackf("parsing schema: %w", err)
			}

			out.xmlSchema = schema
		}
	}

	return &out, nil
}

func (o *FileOutput) GetID() string {
	return o.ID
}

func (o *FileOutput) ClearID() {
	o.ID = ""
}

func (o *FileOutput) GetType() string {
	return o.Type
}

func (o *FileOutput) Run(data any, w io.Writer) error {
	marshalled, err := o.generateOutput(data)
	if err != nil {
		return err
	}

	// Write to disk
	_, err = w.Write(marshalled)
	return err
}

func (o *FileOutput) generateOutput(content any) ([]byte, error) {
	var output []byte
	var err error
	var data []byte
	var dataErr error

	switch o.Format {
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
		return nil, yerr.WithStackf("output type %s not supported", o.Type)
	}

	if err != nil {
		return nil, yerr.WithStackf(
			"marshalling to %s: %w",
			o.Type,
			err,
		)
	}

	if dataErr != nil {
		return nil, yerr.WithStackf("marshalling data to %s: %w", o.Type, dataErr)
	}

	if data == nil {
		data = output
	}
	if err := o.validateOutputSchema(data); err != nil {
		return nil, err
	}

	return output, nil
}

func (o *FileOutput) validateOutputSchema(data []byte) error {
	if o.Schema == "" {
		return nil
	}

	switch o.Format {
	case "json", "md", "yaml":
		res := o.jsonSchema.ValidateJSON(data)

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

		return o.xmlSchema.Validate(doc)
	default:
		return nil
	}
}
