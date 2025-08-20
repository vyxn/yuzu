// Package kitsu implements the provider interface for kitsu metadata
package kitsu

import "github.com/vyxn/yuzu/internal/standard"

type KitsuComicInfoProvider struct {
	cache map[string]MangaInfo
}

func NewKitsuProvider() *KitsuComicInfoProvider {
	return &KitsuComicInfoProvider{cache: map[string]MangaInfo{}}
}

func (p *KitsuComicInfoProvider) ProvideChapter(
	series, chapter string,
) *standard.ComicInfoChapter {
	mangaInfo, ok := p.cache[series]
	if !ok {
		seriesRes := GetSearchByName(series)
		mangaURL := ParseMangaListSelfLink(seriesRes)
		mangaInfoRes := GetURL(mangaURL)
		mangaInfo = ParseMangaInfo(mangaInfoRes)
		p.cache[series] = mangaInfo
	}

	info := GetMangaChapterInfo(mangaInfo.Data.ID, chapter)
	chapterInfo := ParseMangaChapter(info)
	return ParseToComicInfoChapter(mangaInfo, chapterInfo)
}
