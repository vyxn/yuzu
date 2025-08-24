package provider

import (
	"context"
	"reflect"

	"github.com/vyxn/yuzu/internal/standard"
)

func MergedComicInfoChapter(
	ctx context.Context,
	series, chapter string,
	providers ...ComicInfoProvider,
) (*standard.ComicInfoChapter, error) {
	out := &standard.ComicInfoChapter{}

	for _, p := range providers {
		if ci, err := p.ProvideChapter(ctx, series, chapter); err == nil {
			MergeStructs(out, ci)
		} else {
			return nil, err
		}
	}

	return out, nil
}

func MergeStructs(dst, src any) {
	dv := reflect.ValueOf(dst).Elem()
	sv := reflect.ValueOf(src).Elem()

	for i := 0; i < dv.NumField(); i++ {
		df := dv.Field(i)
		sf := sv.Field(i)

		// skip if destination already has a non-zero value
		if !df.IsZero() {
			continue
		}

		// copy value from source
		if df.CanSet() {
			df.Set(sf)
		}
	}
}
