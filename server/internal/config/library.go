package config

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

	"github.com/vyxn/yuzu/internal/library"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

const dirLibraries = "libraries"

var libraryPaths = sync.Map{}

func NewLibrary(filepath string) (p *library.Library, ferr error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, yerr.WithStackf("opening library file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			ferr = errors.Join(ferr, yerr.WithStackf("closing file: %w", err))
		}
	}()

	return library.New(filepath, file)
}

func LoadLibrary(path string) {
	info, err := os.Stat(path)
	if err != nil {
		slog.Warn("could not get path info", slog.String("path", path))
		return
	}

	if info.IsDir() || !slices.Contains(allowedExtensions, filepath.Ext(path)) {
		return
	}

	lib, err := NewLibrary(path)
	if err != nil {
		slog.Warn(
			"skipping library",
			slog.String("reason", "error"),
			slog.String("path", path),
			slog.Any("error", err.Error()),
		)
		return
	}

	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if id == "" || id == string(filepath.Separator) {
		return
	}

	Cfg.Libraries.Store(id, lib)
	providerPaths.Store(path, id)
}

func UnloadLibrary(path string) {
	if id, ok := libraryPaths.LoadAndDelete(path); ok {
		Cfg.Libraries.Delete(id)
		slog.Info("deleted library", slog.String("library", id.(string)))
	}
}

func StoreLibrary(id string, l *library.Library) error {
	subpath := filepath.Join(dirLibraries, id+".json")

	pr, pw := io.Pipe()

	go Cfg.StoreFile(subpath, pr)
	e := json.NewEncoder(pw)
	e.SetIndent("", "  ")
	return e.Encode(l)
}

func DeleteLibrary(id string) error {
	l, ok := Cfg.Libraries.Load(id)
	if !ok {
		return yerr.WithStackf("library with id %q not found", id)
	}

	libs, ok := l.(*library.Library)
	if !ok {
		return yerr.WithStackf("couldn't coerce library with id %q", id)
	}

	if err := os.Remove(libs.ConfigPath()); err != nil {
		return yerr.WithStackf("couldn't remove library %q: %w", id, err)
	}

	// Clear cached values
	Cfg.Libraries.Delete(id)
	libraryPaths.Delete(libs.ConfigPath())
	return nil
}

func LoadLibraries() error {
	return Cfg.GetFiles(
		dirLibraries,
		func(l string, d fs.DirEntry, err error) error {
			if err != nil {
				slog.Warn("walking libraries dir", slog.Any("err", err))
				return nil
			}

			LoadLibrary(l)
			return nil
		},
	)
}

func WatchLibraries(ctx context.Context) {
	for _, dir := range Cfg.Paths {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}

		d := filepath.Join(dir, dirLibraries)
		go Watch(ctx, d, LoadLibrary, UnloadLibrary)
	}
}
