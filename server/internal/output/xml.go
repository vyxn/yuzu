package output

import (
	"bytes"
	"encoding/xml"
	"maps"
	"slices"
)

func encodeValue(enc *xml.Encoder, key string, v any) error {
	start := xml.StartElement{Name: xml.Name{Local: key}}

	switch val := v.(type) {
	case map[string]any:
		// Open tag
		if err := enc.EncodeToken(start); err != nil {
			return err
		}

		keys := slices.Collect(maps.Keys(val))
		slices.Sort(keys)
		for _, k := range keys {
			inner := val[k]
			if err := encodeValue(enc, k, inner); err != nil {
				return err
			}
		}
		// Close tag
		if err := enc.EncodeToken(start.End()); err != nil {
			return err
		}
	case []any:
		for _, item := range val {
			if err := encodeValue(enc, key, item); err != nil {
				return err
			}
		}
	default:
		if err := enc.EncodeElement(val, start); err != nil {
			return err
		}
	}

	return nil
}

func MapToXML(v any) ([]byte, error) {
	buf := &bytes.Buffer{}

	if _, err := buf.WriteString(xml.Header); err != nil {
		return nil, err
	}

	enc := xml.NewEncoder(buf)
	enc.Indent("", "  ")

	switch rootVal := v.(type) {
	case map[string]any:
		keys := slices.Collect(maps.Keys(rootVal))
		slices.Sort(keys)
		for _, k := range keys {
			inner := rootVal[k]
			if err := encodeValue(enc, k, inner); err != nil {
				return nil, err
			}
		}
		// default:
		// 	// if it’s not a map, just encode directly as content
		// 	if err := enc.EncodeElement(rootVal, start); err != nil {
		// 		return nil, err
		// 	}
	}

	if err := enc.Flush(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
