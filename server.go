package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintln(w, "Method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var req CompareRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, `api usage: GET /compare?actual={...}&expected={...}`)
		return
	}
	fmt.Printf("got a request with %s and %s", req.Expected, req.Actual)

	var actual, expected interface{}
	if err := json.Unmarshal(req.Actual, &actual); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Invalid 'actual' JSON")
		return
	}
	if err := json.Unmarshal(req.Expected, &expected); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Invalid 'expected' JSON")
		return
	}

	comparison := compareJSON(actual, expected)

	// Print colored comparison tree to response
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "==== COMPARISON TREE ====")
	printComparison(w, comparison, "", true, true)

	hasMismatch := checkForMismatches(comparison)
	if hasMismatch {
		fmt.Fprintln(w, "\n❌ Differences found")
	} else {
		fmt.Fprintln(w, "\n✅ Objects match")
	}

	url := create_url(req.Actual, req.Expected)
	fmt.Fprintln(w, url)

}
