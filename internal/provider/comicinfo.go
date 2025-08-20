// Package provider sets up interfaces to get metadata from different sources
package provider

import "github.com/vyxn/yuzu/internal/standard"

type ComicInfoProvider interface {
	ProvideChapter(series, chapter string) *standard.ComicInfoChapter
}
