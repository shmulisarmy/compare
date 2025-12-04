package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
)

type ComparisonResult struct {
	Type         string                       `json:"type"` // "match", "mismatch", "object", "array"
	Value        interface{}                  `json:"value,omitempty"`
	Expected     interface{}                  `json:"expected,omitempty"`
	Actual       interface{}                  `json:"actual,omitempty"`
	ExpectedType string                       `json:"expected_type,omitempty"`
	ActualType   string                       `json:"actual_type,omitempty"`
	ExpectedSize int                          `json:"expected_size,omitempty"`
	ActualSize   int                          `json:"actual_size,omitempty"`
	Children     map[string]*ComparisonResult `json:"children,omitempty"`
	ArrayItems   []*ComparisonResult          `json:"array_items,omitempty"`
}

func compareJSON(actual, expected interface{}) *ComparisonResult {
	// Handle nil cases
	if actual == nil && expected == nil {
		return &ComparisonResult{Type: "match", Value: nil}
	}
	if actual == nil || expected == nil {
		return &ComparisonResult{
			Type:         "mismatch",
			Expected:     expected,
			Actual:       actual,
			ExpectedType: getTypeName(expected),
			ActualType:   getTypeName(actual),
		}
	}

	// Get the actual Go types
	actualType := reflect.TypeOf(actual).Kind()
	expectedType := reflect.TypeOf(expected).Kind()

	// Check if types match - be strict about string vs number
	if actualType != expectedType {
		return &ComparisonResult{
			Type:         "mismatch",
			Expected:     expected,
			Actual:       actual,
			ExpectedType: getTypeName(expected),
			ActualType:   getTypeName(actual),
		}
	}

	switch exp := expected.(type) {
	case map[string]interface{}:
		return compareObjects(actual.(map[string]interface{}), exp)
	case []interface{}:
		return compareArrays(actual.([]interface{}), exp)
	default:
		if reflect.DeepEqual(actual, expected) {
			return &ComparisonResult{Type: "match", Value: actual}
		}
		return &ComparisonResult{
			Type:     "mismatch",
			Expected: expected,
			Actual:   actual,
		}
	}
}

// getTypeName returns a human-readable type name
func getTypeName(v interface{}) string {
	if v == nil {
		return "nil"
	}
	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// compareObjects compares two JSON objects
func compareObjects(actual, expected map[string]interface{}) *ComparisonResult {
	result := &ComparisonResult{
		Type:     "object",
		Children: make(map[string]*ComparisonResult),
	}

	allKeys := make(map[string]bool)
	for k := range actual {
		allKeys[k] = true
	}
	for k := range expected {
		allKeys[k] = true
	}

	for key := range allKeys {
		actualVal, actualExists := actual[key]
		expectedVal, expectedExists := expected[key]

		if !actualExists {
			result.Children["missing::"+key] = &ComparisonResult{
				Type:     "mismatch",
				Expected: expectedVal,
				Actual:   nil,
			}
		} else if !expectedExists {
			result.Children["extra::"+key] = &ComparisonResult{
				Type:     "mismatch",
				Expected: nil,
				Actual:   actualVal,
			}
		} else {
			result.Children[key] = compareJSON(actualVal, expectedVal)
		}
	}

	return result
}

// compareArrays compares two JSON arrays
func compareArrays(actual, expected []interface{}) *ComparisonResult {
	if len(actual) != len(expected) {
		return &ComparisonResult{
			Type:         "mismatch",
			ExpectedSize: len(expected),
			ActualSize:   len(actual),
		}
	}

	result := &ComparisonResult{
		Type:       "array",
		ArrayItems: make([]*ComparisonResult, len(expected)),
	}

	for i := 0; i < len(expected); i++ {
		result.ArrayItems[i] = compareJSON(actual[i], expected[i])
	}

	return result
}

// printComparison prints the comparison result as a tree
func printComparison(w io.Writer, result *ComparisonResult, prefix string, isLast bool, useColor bool) {
	connector := "├── "
	extender := "│"
	if isLast {
		connector = "└── "
		extender = " "
	}

	switch result.Type {
	case "match":
		// Matches are shown inline with their parent
		return

	case "mismatch":
		if result.ExpectedType != "" || result.ActualType != "" {
			// Type mismatch
			if useColor {
				fmt.Fprintf(w, "%s        %sExpected:%s <%s>\n", prefix, colorCyan, colorReset, result.ExpectedType)
				fmt.Fprintf(w, "%s        %sActual:%s   {%s %v}\n", prefix, colorRed, colorReset,
					strings.ToUpper(result.ActualType), formatValueShort(result.Actual))
				if result.ExpectedType == "nil" {
					fmt.Fprintf(w, "%s        %sExpected is nil, actual is not%s\n", prefix, colorYellow, colorReset)
				} else {
					fmt.Fprintf(w, "%s        %sType mismatch: expected %s, got %s%s\n", prefix, colorYellow, result.ExpectedType, result.ActualType, colorReset)
				}
			} else {
				fmt.Fprintf(w, "%s        Expected: <%s>\n", prefix, result.ExpectedType)
				fmt.Fprintf(w, "%s        Actual:   {%s %v}\n", prefix, strings.ToUpper(result.ActualType), formatValueShort(result.Actual))
				if result.ExpectedType == "nil" {
					fmt.Fprintf(w, "%s        Expected is nil, actual is not\n", prefix)
				} else {
					fmt.Fprintf(w, "%s        Type mismatch: expected %s, got %s\n", prefix, result.ExpectedType, result.ActualType)
				}
			}
		} else if result.ExpectedSize > 0 || result.ActualSize > 0 {
			// Size mismatch
			if useColor {
				fmt.Fprintf(w, "%s        %sExpected size:%s %d\n", prefix, colorCyan, colorReset, result.ExpectedSize)
				fmt.Fprintf(w, "%s        %sActual size:%s   %d\n", prefix, colorRed, colorReset, result.ActualSize)
			} else {
				fmt.Fprintf(w, "%s        Expected size: %d\n", prefix, result.ExpectedSize)
				fmt.Fprintf(w, "%s        Actual size:   %d\n", prefix, result.ActualSize)
			}
		} else {
			// Value mismatch
			if useColor {
				fmt.Fprintf(w, "%s        %sExpected:%s %v\n", prefix, colorCyan, colorReset, formatValue(result.Expected))
				fmt.Fprintf(w, "%s        %sActual:%s   %v\n", prefix, colorRed, colorReset, formatValue(result.Actual))
			} else {
				fmt.Fprintf(w, "%s        Expected: %v\n", prefix, formatValue(result.Expected))
				fmt.Fprintf(w, "%s        Actual:   %v\n", prefix, formatValue(result.Actual))
			}
		}

	case "object":
		// Sort keys for consistent output
		keys := make([]string, 0, len(result.Children))
		for k := range result.Children {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, key := range keys {
			childIsLast := i == len(keys)-1
			childPrefix := prefix + extender

			displayKey := key
			isMissing := strings.HasPrefix(key, "missing::")
			isExtra := strings.HasPrefix(key, "extra::")

			if isMissing {
				displayKey = strings.TrimPrefix(key, "missing::")
			} else if isExtra {
				displayKey = strings.TrimPrefix(key, "extra::")
			}

			child := result.Children[key]

			// Simple leaf value that matches
			if child.Type == "match" {
				if useColor {
					fmt.Fprintf(w, "%s%s%s%s%s %v\n", prefix, connector, colorYellow, displayKey, colorReset, formatValue(child.Value))
				} else {
					fmt.Fprintf(w, "%s%s%s %v\n", prefix, connector, displayKey, formatValue(child.Value))
				}
			} else if child.Type == "mismatch" {
				// Value with mismatch
				if useColor {
					fmt.Fprintf(w, "%s%s%s%s%s\n", prefix, connector, colorRed, displayKey, colorReset)
				} else {
					fmt.Fprintf(w, "%s%s%s\n", prefix, connector, displayKey)
				}
				printComparison(w, child, childPrefix, childIsLast, useColor)
			} else {
				// Nested object or array - always show structure
				if useColor {
					fmt.Fprintf(w, "%s%s%s%s%s\n", prefix, connector, colorBlue, displayKey, colorReset)
				} else {
					fmt.Fprintf(w, "%s%s%s\n", prefix, connector, displayKey)
				}
				printComparison(w, child, childPrefix, childIsLast, useColor)
			}
		}

	case "array":
		for i, item := range result.ArrayItems {
			childIsLast := i == len(result.ArrayItems)-1
			childPrefix := prefix + extender

			// Show the item index and value/comparison on same structure level
			if item.Type == "match" {
				if useColor {
					fmt.Fprintf(w, "%s%s%s[%d]:%s %s✓%s %v\n", prefix, connector, colorGray, i, colorReset, colorGreen, colorReset, formatValue(item.Value))
				} else {
					fmt.Fprintf(w, "%s%s[%d]: ✓ %v\n", prefix, connector, i, formatValue(item.Value))
				}
			} else if item.Type == "mismatch" {
				// Show index, then mismatch details indented
				if useColor {
					fmt.Fprintf(w, "%s%s%s[%d]:%s\n", prefix, connector, colorRed, i, colorReset)
				} else {
					fmt.Fprintf(w, "%s%s[%d]:\n", prefix, connector, i)
				}
				printComparison(w, item, childPrefix, childIsLast, useColor)
			} else {
				// Nested structure
				if useColor {
					fmt.Fprintf(w, "%s%s%s[%d]:%s\n", prefix, connector, colorGray, i, colorReset)
				} else {
					fmt.Fprintf(w, "%s%s[%d]:\n", prefix, connector, i)
				}
				printComparison(w, item, childPrefix, childIsLast, useColor)
			}
		}
	}
}

// formatValue formats a value for display
func formatValue(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	case bool:
		return fmt.Sprintf("%v", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// formatValueShort formats a value in a compact way
func formatValueShort(v interface{}) string {
	if v == nil {
		return "nil"
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// parseJSONInput parses JSON from a string or file
func parseJSONInput(input string) (interface{}, error) {
	var data interface{}

	if strings.HasPrefix(input, "/") || strings.HasPrefix(input, "./") || strings.HasPrefix(input, "../") {
		fileData, err := os.ReadFile(input)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON string and could not read as file: %v", err)
		}
		err = json.Unmarshal(fileData, &data)
		if err != nil {
			return nil, fmt.Errorf("file content is not valid JSON: %v", err)
		}
		return data, nil
	}

	// Try to parse as JSON string first
	err := json.Unmarshal([]byte(input), &data)
	if err == nil {
		return data, nil
	}

	// Try to read as file

	// If it's not a file path, try to parse as JSON again (might be a JSON string with special characters)
	err = json.Unmarshal([]byte(input), &data)
	if err != nil {
		return nil, fmt.Errorf("input is not valid JSON: %v, if you meant to provide a file path, please ensure it starts with /, ./, or ../", err)
	}

	return data, nil
}

// CompareRequest represents the HTTP request body for comparison
type CompareRequest struct {
	Actual   json.RawMessage `json:"actual"`
	Expected json.RawMessage `json:"expected"`
}

// handleCompare handles the HTTP comparison endpoint
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
	{

		actualURL := url.QueryEscape(string(req.Actual))
		expectedURL := url.QueryEscape(string(req.Expected))
		url := fmt.Sprintf("https://json-diff-pro-ae6c6454.base44.app/?expected=%s&actual=%s", expectedURL, actualURL)
		short_url := func(site string, alias string) string {
			return fmt.Sprintf("\u001B]8;;%s\u0007%s\u001B]8;;\u0007", site, alias)
		}
		fmt.Fprintln(w, "interactive json viewer: "+short_url(url, "json-diff-pro"))
	}
}

// checkForMismatches recursively checks if there are any mismatches
func checkForMismatches(result *ComparisonResult) bool {
	if result.Type == "mismatch" {
		return true
	}
	if result.Children != nil {
		for _, child := range result.Children {
			if checkForMismatches(child) {
				return true
			}
		}
	}
	if result.ArrayItems != nil {
		for _, item := range result.ArrayItems {
			if checkForMismatches(item) {
				return true
			}
		}
	}
	return false
}
