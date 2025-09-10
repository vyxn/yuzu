package library

import (
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
)

type Selectors []*Selector

type Selector struct {
	Type     string         `json:"type"`
	Regex    string         `json:"regex"`
	Captures map[string]int `json:"captures"`
	r        *regexp.Regexp `json:"-"`
}

func (s *Selector) UnmarshalJSON(data []byte) error {
	// shadow type to avoid recursion
	type Alias Selector
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if s.Regex != "" {
		re, err := regexp.Compile(s.Regex)
		if err != nil {
			return fmt.Errorf("invalid regex %q: %w", s.Regex, err)
		}
		s.r = re

		limit := re.NumSubexp()
		for k, i := range s.Captures {
			if i > limit {
				return fmt.Errorf(
					"invalid capture index on key %q: regex limit %d",
					k,
					limit,
				)
			}
		}
	}

	return nil
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

	matches := s.r.FindStringSubmatch(input)
	if matches == nil {
		return nil
	}

	env := map[string]string{}
	for k, i := range s.Captures {
		env[k] = matches[i]
	}
	return env
}
