// Package config handles how to fetch all the service config from disk / db
package config

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

const AppName = "yuzu"

var Cfg *Config

type Config struct {
	IsDev     bool
	Paths     []string
	Providers sync.Map
}

func init() {
	Cfg = new()
}

func new() *Config {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}
	isDev := appEnv == "development"
	return &Config{
		IsDev: isDev,
		Paths: configPaths(isDev),
	}
}

func (cfg *Config) GetFiles(subdir string, fn fs.WalkDirFunc) error {
	var errs error

	for _, d := range cfg.Paths {
		basePath := filepath.Join(d, subdir)

		err := filepath.WalkDir(basePath, fn)
		errors.Join(errs, yerr.WithStackf("walking dir \"%s\": %w", d, err))
	}

	return errs
}

func (cfg *Config) GetFile(subpath string) (string, error) {
	for _, d := range cfg.Paths {
		path := filepath.Join(d, subpath)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", yerr.WithStackf("no config file \"%s\" found", subpath)
}

// func (cfg *Config) EnsureFile(subpath string) (string, error) {
// 	for _, d := range cfg.Paths {
// 		dir := filepath.Join(d, filepath.Dir(subpath))
// 		if fi, err := os.Stat(dir); err != nil {
// 			err := os.MkdirAll(dir, 0700)
// 			if err != nil {
//
// 			}
// 		}
// 		break
// 	}
// }

func configPaths(isDev bool) []string {
	paths := []string{}

	if isDev {
		paths = append(paths, "config")
	}

	home, err := os.UserHomeDir()
	assert.Assert(err == nil, "home env is not set")

	switch runtime.GOOS {
	case "linux", "darwin":
		xdgHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgHome == "" {
			xdgHome = filepath.Join(home, ".config")
		}

		paths = append(paths, filepath.Join(xdgHome, AppName))
		paths = append(paths, path.Join("/etc/xdg", AppName))
		paths = append(paths, path.Join("/etc", AppName))
	case "windows":
		appData := os.Getenv("APPDATA")
		localAppData := os.Getenv("LOCALAPPDATA")
		programData := os.Getenv("PROGRAMDATA")
		if appData != "" {
			paths = append(paths, filepath.Join(appData, AppName))
		}
		if localAppData != "" {
			paths = append(paths, filepath.Join(localAppData, AppName))
		}
		if programData != "" {
			paths = append(paths, filepath.Join(programData, AppName))
		}
	}

	return paths
}
