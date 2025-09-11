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

	"github.com/vyxn/yuzu/internal/library"
	"github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

var allowedExtensions = []string{".json"}

func NewLibraryFromPath(path string) (lib *library.Library, ferr error) {
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

	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	l, err := library.NewLibrary(id, file)
	if err != nil {
		return nil, yerr.WithStackf("unmarshaling library %q: %w", path, err)
	}

	return l, nil
}

type FileLibraryRepository struct {
	subdir      string
	configPaths []string
	libraries   sync.Map
	paths       sync.Map
	idToPath    sync.Map
}

func NewFileLibraryRepository(
	ctx context.Context,
	subdir string,
	configPaths []string,
) (Repository[*library.Library, string], error) {
	r := &FileLibraryRepository{
		subdir:      subdir,
		configPaths: configPaths,
		libraries:   sync.Map{},
		paths:       sync.Map{},
		idToPath:    sync.Map{},
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

	r.libraries.Store(lib.Id, lib)
	r.paths.Store(path, lib)
	r.idToPath.Store(lib.Id, path)
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

		d := filepath.Join(dir, r.subdir)
		go Watch(ctx, d, r.load, r.unload)
	}
}

func (r *FileLibraryRepository) GetAll() ([]*library.Library, error) {
	libs := []*library.Library{}

	r.libraries.Range(func(key any, value any) bool {
		lib, ok := value.(*library.Library)
		assert.Assert(ok, "unexpected type on libraries sync.Map")

		libs = append(libs, lib)
		return true
	})

	return libs, nil
}

func (r *FileLibraryRepository) Get(id string) (*library.Library, error) {
	l, ok := r.libraries.Load(id)
	if !ok {
		return nil, yerr.WithStackf("library %q not found", id)
	}

	lib, ok := l.(*library.Library)
	assert.Assert(ok, "unexpected type on libraries sync.Map")

	return lib, nil
}

func (r *FileLibraryRepository) Save(lib *library.Library) error {
	id := lib.Id
	subpath := filepath.Join(r.subdir, id+".json")

	pr, pw := io.Pipe()

	errc := make(chan error, 1)
	go func() {
		defer close(errc)

		for _, d := range r.configPaths {
			path := filepath.Join(d, subpath)

			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				errc <- yerr.WithStackf("creating required dir %q: %w", d, err)
				return
			}

			f, err := os.Create(path)
			if err != nil {
				errc <- yerr.WithStackf("creating file %q: %w", path, err)
				return
			}
			defer func() {
				if err := f.Close(); err != nil {
					errc <- yerr.WithStackf("closing file %q: %w", path, err)
					return
				}
			}()

			if _, err = io.Copy(f, pr); err != nil {
				errc <- yerr.WithStackf("writing file %q: %w", path, err)
				return
			}

			r.libraries.Store(id, lib)
			r.paths.Store(path, lib)
			r.idToPath.Store(id, path)
			break
		}
	}()

	lib.Id = ""
	e := json.NewEncoder(pw)
	e.SetIndent("", "  ")
	err := e.Encode(lib)

	for werr := range errc {
		err = errors.Join(err, werr)
	}

	return err
}

func (r *FileLibraryRepository) Delete(id string) error {
	p, ok := r.idToPath.Load(id)
	if !ok {
		return yerr.WithStackf("library %q not found", id)
	}

	path, ok := p.(string)
	assert.Assert(ok, "unexpected type on libraries sync.Map")

	if err := os.Remove(path); err != nil {
		return yerr.WithStackf("couldn't remove library %q: %w", id, err)
	}

	r.libraries.Delete(id)
	r.paths.Delete(path)
	r.idToPath.Delete(id)
	return nil
}
