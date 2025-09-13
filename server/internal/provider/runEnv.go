package provider

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

type RunEnv struct {
	secret map[string]string
	public map[string]string
}

func NewRunEnv(public map[string]string, secret map[string]string) *RunEnv {
	pub := make(map[string]string)
	maps.Copy(pub, public)
	sec := make(map[string]string)
	maps.Copy(sec, secret)
	return &RunEnv{public: pub, secret: sec}
}

func (e *RunEnv) String() string {
	return fmt.Sprintf("%+v", e.public)
}

func (e *RunEnv) Replace(input string) string {
	replaced := input
	for k, v := range e.secret {
		replaced = strings.ReplaceAll(replaced, k, v)
	}
	for k, v := range e.public {
		replaced = strings.ReplaceAll(replaced, k, v)
	}

	return replaced
}

func (e *RunEnv) ReplaceAny(input any) any {
	switch val := input.(type) {
	case string:
		return e.Replace(val)
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, inner := range val {
			out[k] = e.ReplaceAny(inner) // recurse
		}
		return out
	case []any: // also handle slices if needed
		out := make([]any, len(val))
		for i, inner := range val {
			out[i] = e.ReplaceAny(inner)
		}
		return out
	default:
		// int, bool, float64, structs, etc.
		return val
	}
}

func (e *RunEnv) AddPublic(key string, value any) {
	switch v := value.(type) {
	case string:
		e.public[key] = v
	case float64:
		e.public[key] = strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			e.public[key] = "true"
		} else {
			e.public[key] = "false"
		}
	// TODO perhaps this does not make sense if we move to map[string]any
	case nil:
		e.public[key] = ""
	default:
		panic(yerr.WithStackf("unsupported type <%v>", v))
	}
}

func (e *RunEnv) AddSecret(key string, value any) {
	switch v := value.(type) {
	case string:
		e.secret[key] = v
	case float64:
		e.secret[key] = strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			e.secret[key] = "true"
		} else {
			e.secret[key] = "false"
		}
	// TODO perhaps this does not make sense if we move to map[string]any
	case nil:
		e.secret[key] = ""
	default:
		panic(yerr.WithStackf("unsupported type <%v>", v))
	}
}
