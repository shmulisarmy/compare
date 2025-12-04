package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

func to_terminal_safe_json_string(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func Compare(expected, actual any, url string) (int, string, error) {
	if info, err := exec.LookPath("compare"); err == nil && info != "" {
		// Use local compare binary

		terminal_command := fmt.Sprintf("compare -expected='%s' -actual='%s'", to_terminal_safe_json_string(expected), to_terminal_safe_json_string(actual))
		cmd := exec.Command("sh", "-c", terminal_command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return 1, string(output), err
		}
		return 0, string(output), nil
	} else {
		println("compare is not a command")
	}
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
