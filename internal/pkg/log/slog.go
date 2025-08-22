package log

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/url"
	"os"

	"github.com/fatih/color"
)

type PrettyHandlerOptions struct {
	SlogOpts slog.HandlerOptions
}

type PrettyHandler struct {
	slog.Handler
	l *log.Logger
}

func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String() + ":"

	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	fields := make(map[string]any, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()

		return true
	})
	for k, v := range fields {
		if s, ok := v.(string); ok {
			if decoded, err := url.QueryUnescape(s); err == nil {
				fields[k] = decoded
			}
		}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(fields); err != nil {
		return err
	}
	b := bytes.TrimRight(buf.Bytes(), "\n")

	timeStr := r.Time.Format("[15:05:05.000]")
	msg := color.CyanString(r.Message)

	h.l.Println(timeStr, level, msg, color.WhiteString(string(b)))

	return nil
}

func NewPrettyHandler(
	out io.Writer,
	opts PrettyHandlerOptions,
) *PrettyHandler {
	h := &PrettyHandler{
		Handler: slog.NewJSONHandler(out, &opts.SlogOpts),
		l:       log.New(out, "", 0),
	}

	return h
}

func NewLogger() *slog.Logger {
	opts := PrettyHandlerOptions{
		SlogOpts: slog.HandlerOptions{
			Level: slog.LevelInfo,
		},
	}
	handler := NewPrettyHandler(os.Stdout, opts)
	return slog.New(handler)
}
