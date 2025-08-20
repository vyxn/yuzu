package provider

import (
	"reflect"

	"github.com/vyxn/yuzu/internal/standard"
)

func MergedComicInfoChapter(
	series, chapter string,
	providers ...ComicInfoProvider,
) *standard.ComicInfoChapter {
	out := &standard.ComicInfoChapter{}

	for _, p := range providers {
		MergeStructs(out, p.ProvideChapter(series, chapter))
	}

	return out
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
