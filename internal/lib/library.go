package lib

import (
	"fmt"
	"os"
	"path"
	"regexp"

	"github.com/vyxn/yuzu/internal/kitsu"
)

var re = regexp.MustCompile(`(?i)^.*?(?:chapter|ch|c)?\s?(\d+).*\.cbz$`)

func Process(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.Type().IsDir() {
			processSeries(path.Join(dir, e.Name()), e.Name())
		}
	}

	return nil
}

func processSeries(dir, seriesName string) error {
	seriesRes := kitsu.GetSearchByName(seriesName)
	mangaUrl := kitsu.ParseMangaListSelfLink(seriesRes)
	mangaInfoRes := kitsu.GetUrl(mangaUrl)
	series := kitsu.ParseMangaInfo(mangaInfoRes)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.Type().IsDir() {
			processChapter(dir, series, e.Name())
		}
	}

	return nil
}

func processChapter(dir string, series kitsu.MangaInfo, chapter string) error {
	if path.Ext(chapter) != ".cbz" {
		return nil
	}

	matches := re.FindStringSubmatch(chapter)
	if len(matches) > 1 {
		fmt.Printf(
			"MATCH: %-25s -> Chapter %s -> %+v\n",
			chapter,
			matches[1],
			matches,
		)
		chapterNumber := matches[1]

		info := kitsu.GetMangaChapterInfo(series.Data.ID, chapterNumber)
		chapterInfo := kitsu.ParseMangaChapter(info)
		ci := kitsu.ParseToComicInfoChapter(series, chapterInfo)
		f, err := os.Create(
			path.Join(dir, fmt.Sprintf("%s.ComicInfo.xml", chapterNumber)),
		)
		if err != nil {
			return err
		}

		if err := ci.Encode(f); err != nil {
			return err
		}
	}

	return nil
}
