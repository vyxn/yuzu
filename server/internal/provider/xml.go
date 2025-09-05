package provider

import (
	"bytes"
	"encoding/xml"
)

func encodeMap(enc *xml.Encoder, m map[string]any) error {
	for k, v := range m {
		start := xml.StartElement{Name: xml.Name{Local: k}}
		switch val := v.(type) {
		case map[string]any:
			enc.EncodeToken(start)
			encodeMap(enc, val)
			enc.EncodeToken(start.End())
		case []any:
			for _, item := range val {
				enc.EncodeElement(item, start)
			}
		default:
			enc.EncodeElement(val, start)
		}
	}
	return nil
}

func MapToXML(m map[string]any, root string) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := xml.NewEncoder(buf)

	start := xml.StartElement{Name: xml.Name{Local: root}}
	enc.EncodeToken(start)
	encodeMap(enc, m)
	enc.EncodeToken(start.End())

	enc.Flush()
	return buf.Bytes(), nil
}
