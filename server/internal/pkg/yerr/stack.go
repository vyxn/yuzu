// Package yerr includes custom error helpers for yuzu
package yerr

import (
	"errors"
	"fmt"
	"runtime"
)

type Frame struct {
	Function string
	File     string
	Line     int
}

func (f Frame) String() string {
	return fmt.Sprintf("%s\n\t%s:%d\n", f.Function, f.File, f.Line)
}

func captureStack(skip int) []string {
	const maxFrames = 32
	pc := make([]uintptr, maxFrames)
	n := runtime.Callers(skip, pc)
	frames := runtime.CallersFrames(pc[:n])

	var result []string
	for {
		frame, more := frames.Next()
		result = append(result, Frame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		}.String())
		if !more {
			break
		}
	}
	return result
}

type stackError struct {
	err   error
	stack []string
}

func (e *stackError) Error() string   { return e.err.Error() }
func (e *stackError) Unwrap() error   { return e.err }
func (e *stackError) Stack() []string { return e.stack }

func WithStack(err error) error {
	if err == nil {
		return nil
	}
	return &stackError{
		err:   err,
		stack: captureStack(3),
	}
}

func WithStackf(format string, a ...any) error {
	return &stackError{
		err:   fmt.Errorf(format, a...),
		stack: captureStack(3),
	}
}

func GetStack(err error) []string {
	var se *stackError
	if errors.As(err, &se) {
		return se.Stack()
	}
	return nil
}
