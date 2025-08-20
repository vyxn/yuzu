package assert

import (
	"fmt"
	// "log"
	"log/slog"
	"os"
	"runtime"
	// "os"
)

var assertData map[string]any = map[string]any{}

func AddAssertData(key string, value any) {
	assertData[key] = value
}

func RemoveAssertData(key string) {
	delete(assertData, key)
}

func runAssert(msg string) {
	for k, v := range assertData {
		slog.Error("context", "key", k, "value", v)
	}

	skip := 3
	// capture stack
	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false)
	stack := string(buf[:n])

	fmt.Fprintf(os.Stderr, "ASSERTION FAILED: %s\n", msg)

	// split into lines
	lines := splitLines(stack)

	// trim top frames (2 lines per frame)
	if skip*2 < len(lines) {
		lines = lines[skip*2:]
	}

	for _, l := range lines {
		fmt.Fprintln(os.Stderr, l)
	}
	// panic(msg)
	// log.Fatal(msg)
	os.Exit(1)
}

func splitLines(s string) []string {
	out := []string{}
	curr := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, curr)
			curr = ""
		} else {
			curr += string(r)
		}
	}
	if curr != "" {
		out = append(out, curr)
	}
	return out
}

// TODO: Think about passing around a context for debugging purposes
func Assert(truth bool, msg string) {
	if !truth {
		runAssert(msg)
	}
}

func NoError(err error, msg string) {
	if err != nil {
		slog.Error("NoError#error encountered", "error", err)
		runAssert(msg)
	}
}
