// Package utils includes helper functions that may be re-used across all project
// modules.
package utils

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vyxn/yuzu/internal/pkg/yerr"
)

func GetLocalOrRemoteFile(path string) ([]byte, error) {
	if IsURL(path) {
		data, err := GetURLData(path)
		if err != nil {
			return nil, fmt.Errorf("couldn't get remote schema: %w", err)
		}

		return data, nil
	} else if IsValidFilePath(path) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, yerr.WithStackf("couldn't get local schema: %w", err)
		}

		return data, nil
	}

	return nil, yerr.WithStackf("not a valid path nor url: %s", path)
}

func GetURLData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, yerr.WithStack(err)
	}

	if !HTTPStatusCodeSuccess(resp.StatusCode) {
		return nil, yerr.WithStackf("failed response: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, yerr.WithStack(err)
	}

	return data, nil
}

func HTTPStatusCodeSuccess(code int) bool {
	return code >= http.StatusOK || code < http.StatusMultipleChoices
}

func IsValidFilePath(s string) bool {
	if !IsPath(s) {
		return false
	}

	info, err := os.Stat(s)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func IsPath(s string) bool {
	clean := filepath.Clean(s)
	return clean != "" && clean != "." && clean != string(filepath.Separator)
}

func IsURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	if u.Scheme == "" || u.Host == "" {
		return false
	}

	if !strings.EqualFold(u.Scheme, "http") &&
		!strings.EqualFold(u.Scheme, "https") {
		return false
	}

	return true
}
