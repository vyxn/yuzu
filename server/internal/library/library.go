// Package library contains all functionality to manage the libraries
package library

import (
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
	// "os"
	// "path"
	// "regexp"
	// "github.com/vyxn/yuzu/internal/config"
	// "github.com/vyxn/yuzu/internal/provider"
)

// var re = regexp.MustCompile(`(?i)^.*?(?:chapter|ch|c)?\s?(\d+).*\.cbz$`)

type Library struct {
	Path      string    `json:"path"`
	Selectors Selectors `json:"selectors"`
}

func New(id string, r io.Reader) (*Library, error) {
	var l Library
	d := json.NewDecoder(r)
	if err := d.Decode(&l); err != nil {
		return nil, yerr.WithStackf("unmarshaling library JSON: %w", err)
	}

	return &l, nil
}

func (l *Library) Select() []*Selection {
	res := []*Selection{}

	filepath.WalkDir(l.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			slog.Info("walking", slog.String("path", path))
		}

		if sel := l.Selectors.run(d.IsDir(), path); sel != nil {
			res = append(res, sel)
		}
		return nil
	})

	return res
}

// func Process(dir string) error {
// 	p, ok := config.Cfg.Providers.Load("kitsu")
// 	if !ok {
// 		return fmt.Errorf("do better this error")
// 	}
//
// 	entries, err := os.ReadDir(dir)
// 	if err != nil {
// 		return err
// 	}
//
// 	for _, e := range entries {
// 		if e.Type().IsDir() {
// 			processSeries(
// 				p.(*provider.HTTPProvider),
// 				path.Join(dir, e.Name()),
// 				e.Name(),
// 			)
// 		}
// 	}
//
// 	return nil
// }
//
// func processSeries(p *provider.HTTPProvider, dir, series string) error {
// 	entries, err := os.ReadDir(dir)
// 	if err != nil {
// 		return err
// 	}
//
// 	for _, e := range entries {
// 		if !e.Type().IsDir() {
// 			processChapter(p, dir, series, e.Name())
// 		}
// 	}
//
// 	return nil
// }
//
// func processChapter(
// 	p *provider.HTTPProvider,
// 	dir, series, chapter string,
// ) error {
// 	if path.Ext(chapter) != ".cbz" {
// 		return nil
// 	}
//
// 	matches := re.FindStringSubmatch(chapter)
// 	if len(matches) > 1 {
// 		fmt.Printf(
// 			"MATCH: %-25s -> Chapter %s -> %+v\n",
// 			chapter,
// 			matches[1],
// 			matches,
// 		)
// 		chapterNumber := matches[1]
//
// 		ci, err := p.Run(
// 			map[string]string{"series": series, "chapter": chapterNumber},
// 		)
// 		if err != nil {
// 			return err
// 		}
//
// 		f, err := os.Create(
// 			path.Join(dir, fmt.Sprintf("%s.ComicInfo.xml", chapterNumber)),
// 		)
// 		if err != nil {
// 			return err
// 		}
//
// 		if _, err := f.Write(ci); err != nil {
// 			return err
// 		}
// 	}
//
// 	return nil
// }
