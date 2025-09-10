package library

import (
	"maps"
	"path/filepath"
	"regexp"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
	"github.com/vyxn/yuzu/internal/repository"
)

type Selectors []*Selector

type Selector struct {
	Type     string
	Regex    string
	Captures map[string]int
	re       *regexp.Regexp
}

func NewSelectors(s repository.Selectors) (Selectors, error) {
	selectors := []*Selector{}
	for _, sel := range s {
		re, err := regexp.Compile(sel.Regex)
		if err != nil {
			return nil, yerr.WithStackf("invalid regex %q: %w", sel.Regex, err)
		}

		limit := re.NumSubexp()
		for k, i := range sel.Captures {
			if i > limit {
				return nil, yerr.WithStackf(
					"invalid capture index on key %q: regex limit %d",
					k, limit,
				)
			}
		}

		selectors = append(selectors, &Selector{
			Type:     sel.Type,
			Regex:    sel.Regex,
			Captures: sel.Captures,
			re:       re,
		})
	}

	return selectors, nil
}

type Selection struct {
	Path  string
	IsDir bool
	Env   map[string]string
}

var dirSelections = map[string]map[string]string{}

func NewSelection(path string, isDir bool) *Selection {
	return &Selection{Path: path, IsDir: isDir, Env: map[string]string{}}
}

func (ss Selectors) run(isDir bool, path string) *Selection {
	sel := NewSelection(path, isDir)
	for _, s := range ss {
		if (isDir && s.Type == "file") || (!isDir && s.Type == "dir") {
			continue
		}

		if captured := s.run(isDir, path); captured != nil {
			maps.Copy(sel.Env, captured)
		}
	}

	if len(sel.Env) == 0 {
		return nil
	}

	if isDir {
		dirSelections[path] = sel.Env
	} else {
		maps.Copy(sel.Env, dirSelections[filepath.Dir(path)])
	}

	return sel
}

func (s *Selector) run(isDir bool, path string) map[string]string {
	input := ""
	switch s.Type {
	case "file":
		if isDir {
			return nil
		}
		input = filepath.Base(path)
	case "dir":
		if !isDir {
			return nil
		}
		input = filepath.Dir(path + string(filepath.Separator))
	default:
		return nil
	}

	matches := s.re.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}

	env := map[string]string{}
	for k, i := range s.Captures {
		env[k] = matches[i]
	}
	return env
}
