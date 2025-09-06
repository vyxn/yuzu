package config

import (
	"io/fs"
	"log/slog"
	"os"
	"path"
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

func Load() error {
	return Cfg.GetFiles(
		dirProviders,
		func(p string, d fs.DirEntry, err error) error {
			// if err != nil && p == basePath {
			// 	return yerr.WithStackf("reading providers dir: %w", err)
			// } else
			if err != nil {
				slog.Warn("walking providers dir", slog.Any("err", err))
				return nil
			}

			if d.IsDir() || !slices.Contains(allowedExtensions, path.Ext(p)) {
				return nil
			}

			prov, err := NewProvider(p)
			if err != nil {
				slog.Warn(
					"skipping provider",
					slog.String("reason", "error"),
					slog.String("file", p),
					slog.Any("error", err),
				)
				return nil
			}

			id := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
			if id == "" || id == string(filepath.Separator) {
				return nil
			}

			if _, ok := Cfg.Providers.Load(id); ok {
				slog.Warn(
					"skipping provider",
					slog.String("reason", "duplicated"),
					slog.String("file", p),
					slog.String("id", id),
				)
				return nil
			}

			Cfg.Providers.Store(id, prov)
			providerPaths.Store(p, id)
			return nil
		},
	)
}
