package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/syncthing/notify"
)

func Watch(ctx context.Context, root string, load, unload func(path string)) {
	c := make(chan notify.EventInfo, 1000)
	defer close(c)

	d := filepath.Join(root, "...")
	if err := notify.Watch(d, c, notify.All); err != nil {
		slog.Warn(
			"could not setup file watcher, please restart the server if config files change",
			slog.Any("error", err),
		)
	}
	defer notify.Stop(c)

	slog.Info(fmt.Sprintf("watching \"%s\"", d))

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

				if info.IsDir() ||
					!slices.Contains(allowedExtensions, filepath.Ext(pth)) {
					continue
				}

				load(pth)

			case notify.Remove, notify.InMovedFrom:
				if !slices.Contains(allowedExtensions, filepath.Ext(e.Path())) {
					continue
				}

				unload(e.Path())
				// case notify.InCloseWrite:
				// 	log.Println("Finished editing:", e.Path())
				// case notify.InMovedTo:
				// 	log.Println("Moved into directory:", e.Path())
			}

			Info()
		}
	}
}
