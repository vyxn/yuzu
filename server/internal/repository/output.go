package repository

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/vyxn/yuzu/internal/output"
	"github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

type FileOutputRepository struct {
	subdir      string
	idCache     sync.Map
	paths       sync.Map
	idToPath    sync.Map
	configPaths []string
}

func NewFileOutputRepository(
	ctx context.Context,
	subdir string,
	configPaths []string,
) (Repository[output.Output, string], error) {
	r := &FileOutputRepository{
		subdir:      subdir,
		idCache:     sync.Map{},
		paths:       sync.Map{},
		configPaths: configPaths,
	}

	err := r.loadAll()
	r.watch(ctx)

	return r, err
}

func (r *FileOutputRepository) loadAll() error {
	var errs error

	for _, d := range r.configPaths {
		basePath := filepath.Join(d, r.subdir)

		err := filepath.WalkDir(
			basePath,
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					// slog.Warn("walking outputs dir", slog.Any("err", err))
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

func (r *FileOutputRepository) load(path string) {
	info, err := os.Stat(path)
	if err != nil {
		slog.Warn("could not get path info", slog.String("path", path))
		return
	}

	if info.IsDir() ||
		!slices.Contains(allowedExtensions, filepath.Ext(path)) {
		return
	}

	out, err := newOutputFromPath(path)
	if err != nil {
		slog.Warn(
			"skipping output",
			slog.String("reason", "error"),
			slog.String("path", path),
			slog.Any("error", err.Error()),
		)
		return
	}

	r.idCache.Store(out.GetID(), out)
	r.paths.Store(path, out)
	r.idToPath.Store(out.GetID(), path)
}

func newOutputFromPath(path string) (prov output.Output, ferr error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, yerr.WithStackf("opening provider %q: %v", path, err)
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
	return output.NewOutput(id, file)
}

func (r *FileOutputRepository) unload(path string) {
	if id, ok := r.paths.LoadAndDelete(path); ok {
		r.idCache.Delete(id)
		r.idToPath.Delete(id)
		slog.Info("deleted output", slog.String("output", id.(string)))
	}
}

func (r *FileOutputRepository) watch(ctx context.Context) {
	for _, dir := range r.configPaths {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}

		d := filepath.Join(dir, r.subdir)
		go Watch(ctx, d, r.load, r.unload)
	}
}

func (r *FileOutputRepository) GetAll() ([]output.Output, error) {
	outs := []output.Output{}
	r.idCache.Range(func(key any, value any) bool {
		outs = append(outs, value.(output.Output))
		return true
	})

	return outs, nil
}

func (r *FileOutputRepository) Get(id string) (output.Output, error) {
	p, ok := r.idCache.Load(id)
	if !ok {
		return nil, yerr.WithStackf("output %q not found", id)
	}

	out, ok := p.(output.Output)
	assert.Assert(ok, "unexpected type on output cache")

	return out, nil
}

func (r *FileOutputRepository) Save(out output.Output) error {
	subpath := filepath.Join(r.subdir, out.GetID()+".json")

	pr, pw := io.Pipe()

	errc := make(chan error, 2)
	go func() {
		defer close(errc)

		for _, d := range r.configPaths {
			path := filepath.Join(d, subpath)

			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				errc <- yerr.WithStackf("creating dir %q: %w", d, err)
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

			id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

			r.idCache.Store(id, out)
			r.paths.Store(path, out)
			r.idToPath.Store(id, path)
			break
		}
	}()

	out.ClearID()
	e := json.NewEncoder(pw)
	e.SetIndent("", "  ")
	if err := e.Encode(out); err != nil {
		return yerr.WithStackf("writing json %q: %w", out.GetID(), err)
	}

	err := pw.Close()
	for werr := range errc {
		err = errors.Join(err, werr)
	}

	return err
}

func (r *FileOutputRepository) Delete(id string) error {
	p, ok := r.idToPath.Load(id)
	if !ok {
		return yerr.WithStackf("output %q not found", id)
	}

	path, ok := p.(string)
	assert.Assert(ok, "unexpected output path type")

	if err := os.Remove(path); err != nil {
		return yerr.WithStackf("removing provider %q: %w", id, err)
	}

	r.idCache.Delete(id)
	r.paths.Delete(path)
	r.idToPath.Delete(id)
	return nil
}
