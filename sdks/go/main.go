package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func Compare(expected, actual any, url string) (int, string, error) {
	if url == "" {
		url = "http://localhost:8080/compare"
	}

	payload := map[string]any{
		"expected": expected,
		"actual":   actual,
	}

	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("GET", url, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

func main() {
	_, body, _ := Compare(
		map[string]any{
			"name": "bob",
			"age":  30,
			"friends": []any{
				map[string]any{"name": "charlie"},
				"bob",
				"gil",
			},
		},
		map[string]any{
			"name": "alice",
			"age":  30,
			"friends": []any{
				map[string]any{"name": "charlie2"},
				"bob",
				"gil",
			},
		},
		"",
	)

	fmt.Println(body)
}
