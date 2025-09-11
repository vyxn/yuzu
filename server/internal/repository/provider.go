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

	"github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
	"github.com/vyxn/yuzu/internal/provider"
)

func NewProviderFromPath(path string) (prov provider.Provider, ferr error) {
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
	return provider.NewProvider(id, file)
}

type FileProviderRepository struct {
	subdir      string
	idCache     sync.Map
	paths       sync.Map
	idToPath    sync.Map
	configPaths []string
}

func NewFileProviderRepository(
	ctx context.Context,
	subdir string,
	configPaths []string,
) (Repository[provider.Provider, string], error) {
	r := &FileProviderRepository{
		subdir:      subdir,
		idCache:     sync.Map{},
		paths:       sync.Map{},
		configPaths: configPaths,
	}

	err := r.loadAll()
	r.watch(ctx)

	return r, err
}

// func (r *FileProviderRepository) filename(id string) string {
// 	return filepath.Join(r.subdir, fmt.Sprintf("%s.json", id))
// }

func (r *FileProviderRepository) loadAll() error {
	var errs error

	for _, d := range r.configPaths {
		basePath := filepath.Join(d, r.subdir)

		err := filepath.WalkDir(
			basePath,
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					slog.Warn("walking providers dir", slog.Any("err", err))
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

func (r *FileProviderRepository) load(path string) {
	info, err := os.Stat(path)
	if err != nil {
		slog.Warn("could not get path info", slog.String("path", path))
		return
	}

	if info.IsDir() || !slices.Contains(allowedExtensions, filepath.Ext(path)) {
		return
	}

	prov, err := NewProviderFromPath(path)
	if err != nil {
		slog.Warn(
			"skipping provider",
			slog.String("reason", "error"),
			slog.String("path", path),
			slog.Any("error", err.Error()),
		)
		return
	}

	r.idCache.Store(prov.ID(), prov)
	r.paths.Store(path, prov)
	r.idToPath.Store(prov.ID(), path)
}

func (r *FileProviderRepository) unload(path string) {
	if id, ok := r.paths.LoadAndDelete(path); ok {
		r.idCache.Delete(id)
		r.idToPath.Delete(id)
		slog.Info("deleted provider", slog.String("provider", id.(string)))
	}
}

func (r *FileProviderRepository) watch(ctx context.Context) {
	for _, dir := range r.configPaths {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}

		d := filepath.Join(dir, r.subdir)
		go Watch(ctx, d, r.load, r.unload)
	}
}

func (r *FileProviderRepository) GetAll() ([]provider.Provider, error) {
	provs := []provider.Provider{}
	r.idCache.Range(func(key any, value any) bool {
		provs = append(provs, value.(provider.Provider))
		return true
	})

	return provs, nil
}

func (r *FileProviderRepository) Get(id string) (provider.Provider, error) {
	p, ok := r.idCache.Load(id)
	if !ok {
		return nil, yerr.WithStackf("provider %q not found", id)
	}

	prov, ok := p.(provider.Provider)
	if !ok {
		return nil, yerr.WithStackf("provider %q has wrong type", id)
	}

	return prov, nil
}

func (r *FileProviderRepository) Save(prov provider.Provider) error {
	subpath := filepath.Join(r.subdir, prov.ID()+".json")

	pr, pw := io.Pipe()

	errc := make(chan error, 1)
	go func() {
		defer close(errc)

		for _, d := range r.configPaths {
			path := filepath.Join(d, subpath)

			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
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
			// prov.ID = id
			// prov.configPath = path

			r.idCache.Store(id, prov)
			r.paths.Store(path, prov)
			r.idToPath.Store(id, path)
			break
		}
	}()

	prov.ClearID()
	e := json.NewEncoder(pw)
	e.SetIndent("", "  ")
	err := e.Encode(prov)

	for werr := range errc {
		err = errors.Join(err, werr)
	}

	return err
}

func (r *FileProviderRepository) Delete(id string) error {
	p, ok := r.idToPath.Load(id)
	if !ok {
		return yerr.WithStackf("provider %q not found", id)
	}

	path, ok := p.(string)
	assert.Assert(ok, "unexpected provider path type")

	if err := os.Remove(path); err != nil {
		return yerr.WithStackf("couldn't remove provider %q: %w", id, err)
	}

	r.idCache.Delete(id)
	r.paths.Delete(path)
	r.idToPath.Delete(id)
	return nil
}
