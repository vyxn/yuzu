package lib

import (
	"fmt"
	"os"
	"path"
	"regexp"

	// "github.com/vyxn/yuzu/internal/kitsu"
	"github.com/vyxn/yuzu/internal/provider"
	"github.com/vyxn/yuzu/internal/provider/myanimelist"
)

var re = regexp.MustCompile(`(?i)^.*?(?:chapter|ch|c)?\s?(\d+).*\.cbz$`)

func Process(dir string) error {
	// p := kitsu.NewKitsuProvider()
	p := myanimelist.NewMyAnimeListProvider()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.Type().IsDir() {
			processSeries(p, path.Join(dir, e.Name()), e.Name())
		}
	}

	return nil
}

func processSeries(p provider.ComicInfoProvider, dir, series string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.Type().IsDir() {
			processChapter(p, dir, series, e.Name())
		}
	}

	return nil
}

func processChapter(
	p provider.ComicInfoProvider,
	dir, series, chapter string,
) error {
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

		ci := p.ProvideChapter(series, chapterNumber)
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
