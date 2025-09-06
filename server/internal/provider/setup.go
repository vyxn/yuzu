package provider

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sync"

	"github.com/vyxn/yuzu/internal/pkg/yerr"

	"github.com/syncthing/notify"
)

var allowedExtensions = []string{".json", ".jsonc"}
var providerPaths = sync.Map{}

func loadProvider(path string) {
	p, err := NewProvider(path)
	if err != nil {
		slog.Warn(
			"skipping provider",
			slog.String("reason", "error"),
			slog.String("file", path),
			slog.Any("error", err),
		)
		return
	}

	if p.ID == "" {
		slog.Warn(
			"skipping provider",
			slog.String("reason", "config has no ID"),
			slog.String("file", path),
		)
		return
	}

	// if _, ok := Providers.Swap(p.ID, p); ok {
	// 	slog.Warn(
	// 		"skipping provider",
	// 		slog.String("reason", "duplicated"),
	// 		slog.String("file", path),
	// 		slog.String("id", p.ID),
	// 	)
	// 	return
	// }

	Providers.Store(p.ID, p)
	providerPaths.Store(path, p.ID)
}

func unloadProvider(path string) {
	if id, ok := providerPaths.LoadAndDelete(path); ok {
		Providers.Delete(id)
		slog.Info("deleted provider", slog.String("provider", id.(string)))
	}
}

func Watch(ctx context.Context, root string) {
	c := make(chan notify.EventInfo, 1000)

	if err := notify.Watch(path.Join(root, "..."), c, notify.All); err != nil {
		slog.Warn(
			"could not setup file watcher, please restart the server if config files change",
			slog.Any("error", err),
		)
	}
	defer notify.Stop(c)

	slog.Info("fs watcher online", slog.String("root", root))

	for {
		select {
		case <-ctx.Done():
			return
		case e, isOpen := <-c:
			if !isOpen {
				return
			}

			slog.Info(
				"received fs event",
				slog.String("event", e.Event().String()),
				slog.String("path", e.Path()),
			)

			switch e.Event() {
			case notify.Create, notify.Write, notify.InCloseWrite, notify.InMovedTo:
				pth := e.Path()
				info, err := os.Stat(pth)
				if err != nil {
					slog.Warn("could not get path info", slog.String("path", pth))
					continue
				}

				if info.IsDir() || !slices.Contains(allowedExtensions, path.Ext(pth)) {
					continue
				}

				loadProvider(pth)

			case notify.Remove, notify.InMovedFrom:
				if !slices.Contains(allowedExtensions, path.Ext(e.Path())) {
					continue
				}

				unloadProvider(e.Path())
				// case notify.InCloseWrite:
				// 	log.Println("Finished editing:", e.Path())
				// case notify.InMovedTo:
				// 	log.Println("Moved into directory:", e.Path())
			}

			ps := []string{}
			Providers.Range(func(key, value any) bool {
				ps = append(ps, key.(string))
				return true
			})
			slog.Info("loaded providers", slog.Any("providers", ps))
		}
	}
}

func Setup() error {
	basePath := "config/providers/"

	var jsonFiles []string
	err := filepath.WalkDir(
		basePath,
		func(p string, d fs.DirEntry, err error) error {
			if err != nil && p == basePath {
				return yerr.WithStackf("reading providers dir: %w", err)
			} else if err != nil {
				slog.Warn("walking providers dir", slog.Any("err", err))
				return nil
			}

			if !d.IsDir() && slices.Contains(allowedExtensions, path.Ext(p)) {
				jsonFiles = append(jsonFiles, p)
			}

			return nil
		},
	)
	if err != nil {
		return yerr.WithStackf("walking providers dir: %w", err)
	}

	ids := []string{}
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

		if _, ok := Providers.Load(p.ID); ok {
			slog.Warn(
				"skipping provider",
				slog.String("reason", "duplicated"),
				slog.String("file", f),
				slog.String("id", p.ID),
			)
			continue
		}

		Providers.Store(p.ID, p)
		ids = append(ids, p.ID)
	}

	slog.Info("providers loaded", slog.Any("providers", ids))
	return nil
}

func NewProvider(filePath string) (*HTTPProvider, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, yerr.WithStackf("opening provider file: %v", err)
	}

	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, yerr.WithStackf("reading provider file: %v", err)
	}

	var provider HTTPProvider
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
