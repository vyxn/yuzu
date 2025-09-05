package lib

import (
	"fmt"
	"os"
	"path"
	"regexp"

	"github.com/vyxn/yuzu/internal/provider"
)

var re = regexp.MustCompile(`(?i)^.*?(?:chapter|ch|c)?\s?(\d+).*\.cbz$`)

func Process(dir string) error {
	p := provider.Providers["kitsu"]

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

func processSeries(p *provider.Provider, dir, series string) error {
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
	p *provider.Provider,
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

		ci, err := p.Run(map[string]string{"series": series, "chapter": chapterNumber})
		if err != nil {
			return err
		}

		f, err := os.Create(
			path.Join(dir, fmt.Sprintf("%s.ComicInfo.xml", chapterNumber)),
		)
		if err != nil {
			return err
		}

		if _, err := f.Write(ci); err != nil {
			return err
		}
	}

	return nil
}
