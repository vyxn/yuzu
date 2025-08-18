// Package standard define standard exports of metadata
package standard

import (
	"encoding/xml"
	"io"
)

type ComicInfoChapter struct {
	XMLName xml.Name `xml:"ComicInfo"`
	ID      int      `xml:"id,attr"`
	Title   string   `xml:"Title"`
	Series  string   `xml:"Series"`
	Number  int      `xml:"Number"`
	// Volume  int      `xml:"Volume"`
	PageCount int    `xml:"PageCount"`
	Summary   string `xml:"Summary"`
	Notes     string `xml:"Notes"`
	Manga     string `xml:"Manga"`
}

func (c ComicInfoChapter) Encode(w io.Writer) error {
	e := xml.NewEncoder(w)
	e.Indent("", "  ")

	if err := e.Encode(c); err != nil {
		return err
	}

	return e.Flush()
}
