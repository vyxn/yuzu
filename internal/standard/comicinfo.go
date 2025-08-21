// Package standard define standard exports of metadata
package standard

import (
	"encoding/xml"
	"io"
)

// ComicInfoChapter https://anansi-project.github.io
type ComicInfoChapter struct {
	XMLName         xml.Name `xml:"ComicInfo"`
	Title           string   `xml:"Title"`
	Series          string   `xml:"Series"`
	Number          string   `xml:"Number"`
	Count           int      `xml:"Count,omitempty"`
	Volume          int      `xml:"Volume,omitempty"`
	AlternateSeries string   `xml:"AlternateSeries,omitempty"`
	AlternateNumber string   `xml:"AlternateNumber,omitempty"`
	AlternateCount  int      `xml:"AlternateCount,omitempty"`
	Summary         string   `xml:"Summary,omitempty"`
	Notes           string   `xml:"Notes,omitempty"`
	Year            int      `xml:"Year,omitempty"`
	Month           int      `xml:"Month,omitempty"`
	Day             int      `xml:"Day,omitempty"`
	Writer          string   `xml:"Writer,omitempty"`
	Penciller       string   `xml:"Penciller,omitempty"`
	Inker           string   `xml:"Inker,omitempty"`
	Colorist        string   `xml:"Colorist,omitempty"`
	Letterer        string   `xml:"Letterer,omitempty"`
	CoverArtist     string   `xml:"CoverArtist,omitempty"`
	Editor          string   `xml:"Editor,omitempty"`
	Translator      string   `xml:"Translator,omitempty"`
	Publisher       string   `xml:"Publisher,omitempty"`
	Imprint         string   `xml:"Imprint,omitempty"`
	Genre           string   `xml:"Genre,omitempty"`
	Tags            string   `xml:"Tags,omitempty"`
	Web             string   `xml:"Web,omitempty"`
	PageCount       int      `xml:"PageCount,omitempty"`
	LanguageISO     string   `xml:"LanguageISO,omitempty"`
	Format          string   `xml:"Format,omitempty"`
	BlackAndWhite   string   `xml:"BlackAndWhite,omitempty"`
	Manga           string   `xml:"Manga,omitempty"`
	Characters      string   `xml:"Characters,omitempty"`
	Teams           string   `xml:"Teams,omitempty"`
	Locations       string   `xml:"Locations,omitempty"`
	ScanInformation string   `xml:"ScanInformation,omitempty"`
	StoryArc        string   `xml:"StoryArc,omitempty"`
	StoryArcNumber  string   `xml:"StoryArcNumber,omitempty"`
	SeriesGroup     string   `xml:"SeriesGroup,omitempty"`
	AgeRating       string   `xml:"AgeRating,omitempty"`
	// Pages           []struct `xml:"Pages,omitempty"`
	CommunityRating     float64 `xml:"CommunityRating,omitempty"`
	MainCharacterOrTeam string  `xml:"MainCharacterOrTeam,omitempty"`
	Review              string  `xml:"Review,omitempty"`
	GTIN                string  `xml:"GTIN,omitempty"`
}

func (c ComicInfoChapter) Encode(w io.Writer) error {
	e := xml.NewEncoder(w)
	e.Indent("", "  ")

	if err := e.Encode(c); err != nil {
		return err
	}

	return e.Flush()
}
