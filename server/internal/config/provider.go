package config

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
	"github.com/vyxn/yuzu/internal/provider"
)

const dirProviders = "providers"

var providerPaths = sync.Map{}
var allowedExtensions = []string{".json", ".jsonc"}

func NewProvider(filepath string) (provider.Provider, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, yerr.WithStackf("opening provider file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// TODO
		}
	}()

	return provider.New(filepath, file)
}

func Info() {
	ps := []string{}
	Cfg.Providers.Range(func(key, value any) bool {
		ps = append(ps, key.(string))
		return true
	})
	slog.Info(
		"config",
		slog.Bool("dev", Cfg.IsDev),
		slog.Any("directories", Cfg.Paths),
		slog.Any("providers", ps),
	)
}

func LoadProvider(path string) {
	info, err := os.Stat(path)
	if err != nil {
		slog.Warn("could not get path info", slog.String("path", path))
		return
	}

	if info.IsDir() || !slices.Contains(allowedExtensions, filepath.Ext(path)) {
		return
	}

	prov, err := NewProvider(path)
	if err != nil {
		slog.Warn(
			"skipping provider",
			slog.String("reason", "error"),
			slog.String("path", path),
			slog.Any("error", err),
		)
		return
	}

	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if id == "" || id == string(filepath.Separator) {
		return
	}

	// if _, ok := Cfg.Providers.Load(id); ok {
	// 	slog.Warn(
	// 		"skipping provider",
	// 		slog.String("reason", "duplicated"),
	// 		slog.String("path", path),
	// 		slog.String("id", id),
	// 	)
	// 	return nil
	// }

	Cfg.Providers.Store(id, prov)
	providerPaths.Store(path, id)
}

func UnloadProvider(path string) {
	if id, ok := providerPaths.LoadAndDelete(path); ok {
		Cfg.Providers.Delete(id)
		slog.Info("deleted provider", slog.String("provider", id.(string)))
	}
}

func Load() error {
	return Cfg.GetFiles(
		dirProviders,
		func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				slog.Warn("walking providers dir", slog.Any("err", err))
				return nil
			}

			LoadProvider(p)
			return nil
		},
	)
}

func WatchProviders(ctx context.Context) {
	for _, dir := range Cfg.Paths {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}

		d := filepath.Join(dir, dirProviders)
		go Watch(ctx, d, LoadProvider, UnloadProvider)
	}
}
