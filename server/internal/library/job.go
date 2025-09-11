package library

import (
	"encoding/json"
	"fmt"
	"log/slog"
	// "sync"
	"time"

	// "github.com/vyxn/yuzu/internal/config"

	"github.com/robfig/cron/v3"
)

type Job struct {
	Schedule  string        `json:"schedule"`
	schedule  cron.Schedule `json:"-"`
	Providers []JobProvider `json:"providers"`
	library   *Library      `json:"-"`
}

func (j *Job) UnmarshalJSON(data []byte) error {
	// shadow type to avoid recursion
	type Alias Job
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(j),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if j.Schedule != "" {
		sch, err := cron.ParseStandard(j.Schedule)
		if err != nil {
			return fmt.Errorf("invalid schedule %q: %w", j.Schedule, err)
		}

		j.schedule = sch
	}

	return nil
}

type JobProvider struct {
	ID     string            `json:"id"`
	Inputs map[string]string `json:"inputs"`
}

// Next returns the next time this job would be called after the provided time
func (j *Job) Next(t time.Time) time.Time {
	return j.schedule.Next(t)
}

func (j *Job) Run() {
	slog.Info("executing job", slog.String("library", j.library.Id))

	// selections := j.library.Select()
	// wg := sync.WaitGroup{}
	// for _, provider := range j.Providers {
	// 	prov, ok := config.Cfg.Providers.Load(provider.ID)
	//
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	//
	// 		for _, selection := range selections {
	//
	// 		}
	// 	}()
	// }
	// wg.Wait()
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
