package handler

import (
	"net/url"
	"strings"
)

func queryToMap(queryParams url.Values, sep string) map[string]string {
	m := make(map[string]string)
	for k, v := range queryParams {
		m[k] = strings.Join(v, sep)
	}
	return m
}
