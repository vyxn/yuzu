package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

const dirLibraries = "libraries"

var allowedExtensions = []string{".json"}

type Library struct {
	ID         string    `json:"-"`
	configPath string    `json:"-"`
	Path       string    `json:"path"`
	Selectors  Selectors `json:"selectors"`
	Jobs       []*Job    `json:"jobs"`
}

type Selectors []*Selector

type Selector struct {
	Type     string         `json:"type"`
	Regex    string         `json:"regex"`
	Captures map[string]int `json:"captures"`
}

type Job struct {
	Schedule  string        `json:"schedule"`
	Providers []JobProvider `json:"providers"`
}

type JobProvider struct {
	ID     string            `json:"id"`
	Inputs map[string]string `json:"inputs"`
}

func NewLibrary(id string, r io.Reader) (*Library, error) {
	var l Library
	d := json.NewDecoder(r)
	if err := d.Decode(&l); err != nil {
		return nil, yerr.WithStackf("unmarshaling library %q: %w", id, err)
	}

	l.ID = id
	return &l, nil
}

func NewLibraryFromPath(path string) (lib *Library, ferr error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, yerr.WithStackf("opening library %q: %v", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			ferr = errors.Join(
				ferr,
				yerr.WithStackf("closing file %q: %w", path, err),
			)
		}
	}()

	var l Library
	d := json.NewDecoder(file)
	if err := d.Decode(&l); err != nil {
		return nil, yerr.WithStackf("unmarshaling library %q: %w", path, err)
	}

	l.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	l.configPath = path
	return &l, nil
}

type FileLibraryRepository struct {
	subdir      string
	libraries   sync.Map
	paths       sync.Map
	configPaths []string
}

func NewFileLibraryRepository(
	ctx context.Context,
	subdir string,
	configPaths []string,
) (Repository[*Library, string], error) {
	r := &FileLibraryRepository{
		subdir:      subdir,
		libraries:   sync.Map{},
		paths:       sync.Map{},
		configPaths: configPaths,
	}

	err := r.loadAll()
	r.watch(ctx)

	return r, err
}

func (r *FileLibraryRepository) filename(id string) string {
	return filepath.Join(r.subdir, fmt.Sprintf("%s.json", id))
}

func (r *FileLibraryRepository) loadAll() error {
	var errs error

	for _, d := range r.configPaths {
		basePath := filepath.Join(d, r.subdir)

		err := filepath.WalkDir(
			basePath,
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					slog.Warn("walking libraries dir", slog.Any("err", err))
					return nil
				}

				r.load(path)
				return nil
			},
		)
		errors.Join(errs, yerr.WithStackf("walking dir %q: %w", d, err))
	}

	return errs
}

func (r *FileLibraryRepository) load(path string) {
	info, err := os.Stat(path)
	if err != nil {
		slog.Warn("could not get path info", slog.String("path", path))
		return
	}

	if info.IsDir() || !slices.Contains(allowedExtensions, filepath.Ext(path)) {
		return
	}

	lib, err := NewLibraryFromPath(path)
	if err != nil {
		slog.Warn(
			"skipping library",
			slog.String("reason", "error"),
			slog.String("path", path),
			slog.Any("error", err.Error()),
		)
		return
	}

	r.libraries.Store(lib.ID, lib)
	r.paths.Store(path, lib)
}

func (r *FileLibraryRepository) unload(path string) {
	if id, ok := r.paths.LoadAndDelete(path); ok {
		r.libraries.Delete(id)
		slog.Info("deleted library", slog.String("library", id.(string)))
	}
}

func (r *FileLibraryRepository) watch(ctx context.Context) {
	for _, dir := range r.configPaths {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}

		d := filepath.Join(dir, dirLibraries)
		go Watch(ctx, d, r.load, r.unload)
	}
}

func (r *FileLibraryRepository) GetAll() ([]*Library, error) {
	libs := []*Library{}
	r.libraries.Range(func(key any, value any) bool {
		libs = append(libs, value.(*Library))
		return true
	})

	return libs, nil
}

func (r *FileLibraryRepository) Get(id string) (*Library, error) {
	l, ok := r.libraries.Load(id)
	if !ok {
		return nil, yerr.WithStackf("library %q not found", id)
	}

	lib, ok := l.(*Library)
	if !ok {
		return nil, yerr.WithStackf("library %q has wrong type", id)
	}

	return lib, nil
}

func (r *FileLibraryRepository) Save(lib *Library) error {
	subpath := filepath.Join(dirLibraries, lib.ID+".json")

	pr, pw := io.Pipe()

	errc := make(chan error, 1)
	go func() {
		defer close(errc)

		for _, d := range r.configPaths {
			path := filepath.Join(d, subpath)
			err := os.MkdirAll(filepath.Dir(path), 0700)
			if err != nil {
				errc <- yerr.WithStackf("couldn't create required dir %q: %w", d, err)
				return
			}

			f, err := os.Create(path)
			if err != nil {
				errc <- yerr.WithStackf("couldn't create file %q: %w", path, err)
				return
			}
			defer func() {
				if err := f.Close(); err != nil {
					errc <- yerr.WithStackf("couldn't close file %q: %w", path, err)
					return
				}
			}()

			if _, err = io.Copy(f, pr); err != nil {
				errc <- yerr.WithStackf("couldn't write to file %q: %w", path, err)
				return
			}

			id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			lib.ID = id
			lib.configPath = path

			r.libraries.Store(id, lib)
			r.paths.Store(path, lib)
			break
		}
	}()

	e := json.NewEncoder(pw)
	e.SetIndent("", "  ")
	err := e.Encode(lib)

	for werr := range errc {
		err = errors.Join(err, werr)
	}

	return err
}

func (r *FileLibraryRepository) Delete(id string) error {
	l, ok := r.libraries.Load(id)
	if !ok {
		return yerr.WithStackf("library with id %q not found", id)
	}

	libs, ok := l.(*Library)
	if !ok {
		return yerr.WithStackf("couldn't coerce library with id %q", id)
	}

	if err := os.Remove(libs.configPath); err != nil {
		return yerr.WithStackf("couldn't remove library %q: %w", id, err)
	}

	r.libraries.Delete(id)
	r.paths.Delete(libs.configPath)
	return nil
}
