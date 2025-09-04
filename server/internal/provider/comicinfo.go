// Package provider sets up interfaces to get metadata from different sources
package provider

import (
	"context"

	"github.com/vyxn/yuzu/internal/standard"
)

type ComicInfoProvider interface {
	ProvideChapter(
		ctx context.Context,
		series, chapter string,
	) (*standard.ComicInfoChapter, error)
}
