package main

import (
	"encoding/json"
	"net/url"
	"strings"
)

func URLEncodeJSON(v interface{}) (string, error) {
	var js string

	switch t := v.(type) {
	case string:
		js = t
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		js = string(b)
	}

	// crude check whether value already contains percent-encoding of JSON start
	if looksPercentEncoded(js) {
		return "expected=" + js, nil
	}

	// first encode -> %7B%22name%22...
	first := url.QueryEscape(js)
	// second encode -> %257B%2522name%2522...
	second := url.QueryEscape(first)
	return "expected=" + second, nil
}

func looksPercentEncoded(s string) bool {
	// If string contains "%7B" (encoded '{') or "%25" (encoded '%'), assume already encoded.
	// This is conservative; adjust if you have different heuristics.
	up := strings.ToUpper(s)
	return strings.Contains(up, "%7B") || strings.Contains(up, "%25")
}
