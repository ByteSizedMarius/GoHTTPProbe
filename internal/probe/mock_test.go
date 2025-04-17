package probe

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// mockServer creates a test server for HTTP methods testing
func createMockServer() *httptest.Server {
	allowedMethods := []string{"GET", "POST", "OPTIONS", "HEAD"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add a test cookie for testing cookie jar
		http.SetCookie(w, &http.Cookie{
			Name:  "test-cookie",
			Value: "test-value",
		})

		// Add a custom header for testing
		w.Header().Set("X-Test-Header", "test-value")

		method := r.Method

		// Check if the method is allowed
		allowed := false
		for _, m := range allowedMethods {
			if method == m {
				allowed = true
				break
			}
		}

		// Special case for OPTIONS to return allowed methods
		if method == "OPTIONS" {
			w.Header().Set("Allow", "GET, POST, OPTIONS, HEAD")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Return appropriate status based on method
		if allowed {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Method " + method + " allowed"))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte("Method " + method + " not allowed"))
		}
	}))

	return server
}

func TestRunSingleProbe(t *testing.T) {
	// Create a test server
	server := createMockServer()
	defer server.Close()

	// Create a test logger
	logger := &Logger{
		Verbose: false,
		Quiet:   true,
	}

	// Create a test config
	config := Config{
		URL:         server.URL,
		Verbose:     false,
		Quiet:       true,
		Threads:     2,
		JSONFile:    "",
		Wordlist:    "",
		CookieJar:   "",
		Headers:     []string{"Test-Header: TestValue"},
		Cookies:     "test=value",
		SafeOnly:    false,
		Insecure:    false,
		FollowRedir: true,
	}

	// Run the probe
	err := runSingleProbe(config, logger)
	if err != nil {
		t.Fatalf("Unexpected error running single probe: %v", err)
	}

	// Test with safe mode enabled
	config.SafeOnly = true
	err = runSingleProbe(config, logger)
	if err != nil {
		t.Fatalf("Unexpected error running single probe with safe mode: %v", err)
	}
}

func TestRunWithMultipleURLs(t *testing.T) {
	// Create a test server
	server := createMockServer()
	defer server.Close()

	// Create a temporary file with URLs
	tmpDir := t.TempDir()
	inputFile := tmpDir + "/urls.txt"
	content := server.URL + "\n" + server.URL + "/test\n"
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test URL file: %v", err)
	}

	// Create a test config with input file
	config := Config{
		InputFile: inputFile,
		Verbose:   false,
		Quiet:     true,
		Threads:   2,
		SafeOnly:  true,
	}

	// Run the probe
	err := Run(config)
	if err != nil {
		t.Fatalf("Unexpected error running probe with multiple URLs: %v", err)
	}
}

func TestMethodsHandling(t *testing.T) {
	// Create a test wordlist file
	tmpDir := t.TempDir()
	wordlistFile := tmpDir + "/methods.txt"
	content := "GET\nPOST\nCUSTOM\n# Comment line\nANOTHER-METHOD"
	if err := os.WriteFile(wordlistFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test wordlist file: %v", err)
	}

	logger := &Logger{
		Verbose: true, // Enable verbose for coverage
		Quiet:   false,
	}

	// Test with custom wordlist
	config := Config{
		Wordlist: wordlistFile,
	}

	methods, err := getMethods(config, logger)
	if err != nil {
		t.Fatalf("Unexpected error getting methods: %v", err)
	}

	// Verify some expected methods are present
	expectedMethods := []string{"GET", "POST", "CUSTOM", "ANOTHER-METHOD"}
	for _, expected := range expectedMethods {
		found := false
		for _, m := range methods {
			if m == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected method %q not found in methods list", expected)
		}
	}

	// Test with OPTIONS request
	server := createMockServer()
	defer server.Close()

	optionsConfig := Config{
		URL: server.URL,
	}

	optionsMethods, err := getMethodsFromOptions(optionsConfig, logger)
	if err != nil {
		t.Fatalf("Unexpected error getting methods from OPTIONS: %v", err)
	}

	// Verify expected methods from OPTIONS
	optionsExpected := []string{"GET", "POST", "OPTIONS", "HEAD"}
	for _, expected := range optionsExpected {
		found := false
		for _, m := range optionsMethods {
			if m == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected method %q not found in OPTIONS methods", expected)
		}
	}
}

func TestPrintResultsAndExportJSON(t *testing.T) {
	// Create test results
	results := map[string]Result{
		"GET": {
			StatusCode: 200,
			Length:     100,
			Reason:     "OK",
		},
		"POST": {
			StatusCode: 200,
			Length:     200,
			Reason:     "OK",
		},
		"DELETE": {
			StatusCode: 405,
			Length:     0,
			Reason:     "Method Not Allowed",
		},
		"PUT": {
			StatusCode: 501,
			Length:     0,
			Reason:     "Not Implemented",
		},
	}

	// Create a sorted list of methods
	methods := []string{"DELETE", "GET", "POST", "PUT"}

	// Test printing results
	printResults(methods, results)

	// Test exporting to JSON
	tmpDir := t.TempDir()
	jsonFile := tmpDir + "/results.json"

	err := exportToJSON(jsonFile, results)
	if err != nil {
		t.Fatalf("Unexpected error exporting to JSON: %v", err)
	}

	// Verify the JSON file exists
	if _, err = os.Stat(jsonFile); os.IsNotExist(err) {
		t.Error("Expected JSON file to exist, but it doesn't")
	}

	// Read the JSON file
	content, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}

	// Check that it contains the expected methods
	for _, method := range methods {
		if !contains(string(content), method) {
			t.Errorf("Expected JSON to contain method %q", method)
		}
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
