package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	actualFlag := flag.String("actual", "", "Actual JSON (string or file path)")
	expectedFlag := flag.String("expected", "", "Expected JSON (string or file path)")
	serverFlag := flag.Bool("server", false, "Run as HTTP server")
	portFlag := flag.String("port", "8080", "Server port (only used with --server)")
	noColorFlag := flag.Bool("no-color", false, "Disable colored output")

	flag.Parse()

	if *serverFlag {
		// Run as HTTP server
		http.HandleFunc("/compare", handleCompare)
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, `JSON Comparison API
====================

Endpoint: GET /compare

Request Body:
{
  "actual": {...},
  "expected": {...}
}

Example:
--------
curl -X GET http://localhost:%s/compare \
  -H "Content-Type: application/json" \
  -d '{
    "actual": {"name": "bob", "age": 30},
    "expected": {"name": "alice", "age": 30}
  }'

Response:
---------
Plain text comparison tree with ANSI colors showing differences
`, *portFlag)
		})

		addr := ":" + *portFlag
		fmt.Printf("🚀 JSON Comparison Server\n")
		fmt.Printf("   Listening on http://localhost%s\n", addr)
		fmt.Printf("   POST /compare to compare JSON objects\n\n")
		log.Fatal(http.ListenAndServe(addr, nil))
		return
	}

	// Run as CLI tool
	if *actualFlag == "" || *expectedFlag == "" {
		fmt.Println("JSON Comparison Tool")
		fmt.Println("====================\n")
		fmt.Println("Usage:")
		fmt.Println("  Compare JSON:")
		fmt.Println("    json-compare -actual='{...}' -expected='{...}'")
		fmt.Println("    json-compare -actual=file1.json -expected=file2.json")
		fmt.Println("\n  Options:")
		fmt.Println("    -no-color    Disable colored output")
		fmt.Println("\n  Server mode:")
		fmt.Println("    json-compare --server")
		fmt.Println("    json-compare --server -port=9000")
		fmt.Println("\nExamples:")
		fmt.Println(`  json-compare -actual='{"name":"bob"}' -expected='{"name":"alice"}'`)
		fmt.Println(`  json-compare -actual='{"age":30}' -expected='{"age":"30"}'`)
		fmt.Println(`  json-compare -actual='{"items":[1,2]}' -expected='{"items":["1","2"]}'`)
		fmt.Println("  json-compare -actual=actual.json -expected=expected.json")
		fmt.Println("\nNote: Use double quotes inside JSON strings. Single quotes don't work in JSON.")
		os.Exit(1)
	}

	actual, err := parseJSONInput(*actualFlag)
	if err != nil {
		log.Fatalf("Error parsing actual JSON: %v\n", err)
	}

	expected, err := parseJSONInput(*expectedFlag)
	if err != nil {
		log.Fatalf("Error parsing expected JSON: %v\n", err)
	}

	comparison := compareJSON(actual, expected)

	useColor := !*noColorFlag
	fmt.Println("\n==== COMPARISON TREE ====")
	printComparison(os.Stdout, comparison, "", true, useColor)

	// Check if there are any mismatches
	hasMismatch := checkForMismatches(comparison)
	if hasMismatch {
		if useColor {
			fmt.Printf("\n%s❌ Differences found%s\n", colorRed, colorReset)
		} else {
			fmt.Println("\n❌ Differences found")
		}
		os.Exit(1)
	} else {
		if useColor {
			fmt.Printf("\n%s✅ Objects match%s\n", colorGreen, colorReset)
		} else {
			fmt.Println("\n✅ Objects match")
		}
	}
}
