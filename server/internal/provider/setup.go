package provider

import (
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

func Setup() error {
	basePath := "internal/provider/"

	var jsonFiles []string
	err := filepath.WalkDir(
		basePath,
		func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				slog.Warn("walking file on provider dir", slog.Any("err", err))
			}
			if !d.IsDir() && path.Ext(p) == ".json" {
				jsonFiles = append(jsonFiles, p)
			}
			return nil
		},
	)

	if err != nil {
		return yerr.WithStackf("walking providers dir %w", err)
	}

	for _, f := range jsonFiles {
		p, err := NewProvider(f)
		if err != nil {
			slog.Warn(
				"skipping provider",
				slog.String("reason", "error"),
				slog.String("file", f),
				slog.Any("error", err),
			)
			continue
		}

		if _, ok := Providers[p.ID]; ok {
			slog.Warn(
				"skipping provider",
				slog.String("reason", "duplicated"),
				slog.String("file", f),
				slog.String("id", p.ID),
			)
			continue
		} else {
			Providers[p.ID] = p
		}
	}

	slog.Info(
		"provider module initialized",
		slog.Any("providers", slices.Collect(maps.Keys(Providers))),
	)
	return nil
}

func NewProvider(filePath string) (*Provider, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, yerr.WithStackf("opening provider file: %v", err)
	}

	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, yerr.WithStackf("reading provider file: %v", err)
	}

	var provider Provider
	if err := json.Unmarshal(bytes, &provider); err != nil {
		return nil, yerr.WithStackf("unmarshaling provider JSON: %v", err)
	}

	m := make(map[string]string)
	for env, placeholder := range provider.Envs {
		if v, ok := os.LookupEnv(env); ok {
			m[placeholder] = v
		} else {
			slog.Warn(
				"provider env is not configured",
				slog.String("provider", provider.ID),
				slog.String("env", env),
				slog.String("placeholder", placeholder),
			)
		}
	}
	provider.Envs = m

	return &provider, nil
}
