package probe

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestParseHeaders(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected http.Header
	}{
		{
			name:  "Single header",
			input: []string{"User-Agent: TestAgent"},
			expected: http.Header{
				"User-Agent": []string{"TestAgent"},
			},
		},
		{
			name:  "Multiple headers",
			input: []string{"User-Agent: TestAgent", "Accept: application/json"},
			expected: http.Header{
				"User-Agent": []string{"TestAgent"},
				"Accept":     []string{"application/json"},
			},
		},
		{
			name:  "Comma-separated headers",
			input: []string{"User-Agent: TestAgent, Accept: application/json"},
			expected: http.Header{
				"User-Agent": []string{"TestAgent"},
				"Accept":     []string{"application/json"},
			},
		},
		{
			name:     "Empty input",
			input:    []string{},
			expected: http.Header{},
		},
		{
			name:     "Malformed header",
			input:    []string{"Invalid-Header"},
			expected: http.Header{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseHeaders(tc.input)
			if err != nil {
				t.Fatalf("parseHeaders(%v) returned error: %v", tc.input, err)
			}

			for key, expectedValues := range tc.expected {
				values := result[key]
				if len(values) != len(expectedValues) {
					t.Errorf("Expected %d values for header %s, got %d", len(expectedValues), key, len(values))
					continue
				}

				for i, expectedValue := range expectedValues {
					if values[i] != expectedValue {
						t.Errorf("Expected header %s value %d to be %q, got %q", key, i, expectedValue, values[i])
					}
				}
			}
		})
	}
}

func TestParseHeadersFromFile(t *testing.T) {
	// Create a temporary header file
	tmpDir := t.TempDir()
	headerFile := filepath.Join(tmpDir, "test-headers.txt")
	content := "User-Agent: TestAgent\nAccept: application/json\n# This is a comment\nContent-Type: text/plain"
	if err := os.WriteFile(headerFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test header file: %v", err)
	}

	// Run the test
	result, err := parseHeaders([]string{headerFile})
	if err != nil {
		t.Fatalf("parseHeaders([%q]) returned error: %v", headerFile, err)
	}

	// Verify results
	expected := http.Header{
		"User-Agent":   []string{"TestAgent"},
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"text/plain"},
	}

	for key, expectedValues := range expected {
		values := result[key]
		if len(values) != len(expectedValues) {
			t.Errorf("Expected %d values for header %s, got %d", len(expectedValues), key, len(values))
			continue
		}

		for i, expectedValue := range expectedValues {
			if values[i] != expectedValue {
				t.Errorf("Expected header %s value %d to be %q, got %q", key, i, expectedValue, values[i])
			}
		}
	}
}

func TestParseCookies(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "Single cookie",
			input: "name=value",
			expected: map[string]string{
				"name": "value",
			},
		},
		{
			name:  "Multiple cookies",
			input: "name=value; token=abc123",
			expected: map[string]string{
				"name":  "value",
				"token": "abc123",
			},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "Malformed cookie",
			input:    "invalid-cookie",
			expected: map[string]string{},
		},
		{
			name:  "Cookie with spaces",
			input: "name = value ; token = abc123",
			expected: map[string]string{
				"name":  "value",
				"token": "abc123",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseCookies(tc.input)
			if err != nil {
				t.Fatalf("parseCookies(%q) returned error: %v", tc.input, err)
			}

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d cookies, got %d", len(tc.expected), len(result))
			}

			for key, expectedValue := range tc.expected {
				value, exists := result[key]
				if !exists {
					t.Errorf("Expected cookie %q not found in result", key)
					continue
				}

				if value != expectedValue {
					t.Errorf("Expected cookie %q to have value %q, got %q", key, expectedValue, value)
				}
			}
		})
	}
}

func TestParseCookiesFromFile(t *testing.T) {
	// Create a temporary cookie file
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "test-cookies.txt")
	content := "name=value\ntoken=abc123\n# This is a comment\nsession=xyz789"
	if err := os.WriteFile(cookieFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test cookie file: %v", err)
	}

	// Run the test
	result, err := parseCookies(cookieFile)
	if err != nil {
		t.Fatalf("parseCookies(%q) returned error: %v", cookieFile, err)
	}

	// Verify results
	expected := map[string]string{
		"name":    "value",
		"token":   "abc123",
		"session": "xyz789",
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d cookies, got %d", len(expected), len(result))
	}

	for key, expectedValue := range expected {
		value, exists := result[key]
		if !exists {
			t.Errorf("Expected cookie %q not found in result", key)
			continue
		}

		if value != expectedValue {
			t.Errorf("Expected cookie %q to have value %q, got %q", key, expectedValue, value)
		}
	}
}

func TestGetMethods(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	wordlistFile := filepath.Join(tmpDir, "test-methods.txt")
	content := "GET\nPOST\nCUSTOM\n# This is a comment\nANOTHER"
	if err := os.WriteFile(wordlistFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test wordlist file: %v", err)
	}

	// Setup test config
	config := Config{
		Wordlist: wordlistFile,
		Verbose:  false,
		Quiet:    true,
	}

	logger := &Logger{
		Verbose: config.Verbose,
		Quiet:   config.Quiet,
	}

	// Run the test
	methods, err := getMethods(config, logger)
	if err != nil {
		t.Fatalf("getMethods() returned error: %v", err)
	}

	// Verify the methods list
	// It should contain both the default methods and the ones from our wordlist
	expectedCustomMethods := []string{"ANOTHER", "CUSTOM", "GET", "POST"}
	for _, method := range expectedCustomMethods {
		found := false
		for _, m := range methods {
			if m == method {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected method %q not found in result", method)
		}
	}
}

func TestBuildHTTPClient(t *testing.T) {
	testCases := []struct {
		name          string
		config        Config
		expectError   bool
		checkRedirect bool
	}{
		{
			name: "Default Config",
			config: Config{
				Insecure:    false,
				FollowRedir: true,
				Proxy:       "",
			},
			expectError:   false,
			checkRedirect: false,
		},
		{
			name: "Insecure with No Redirects",
			config: Config{
				Insecure:    true,
				FollowRedir: false,
				Proxy:       "",
			},
			expectError:   false,
			checkRedirect: true,
		},
		{
			name: "Valid Proxy",
			config: Config{
				Insecure:    false,
				FollowRedir: true,
				Proxy:       "http://localhost:8080",
			},
			expectError:   false,
			checkRedirect: false,
		},
		{
			name: "Invalid Proxy",
			config: Config{
				Insecure:    false,
				FollowRedir: true,
				Proxy:       "://not-a-valid-url", // This should cause an error
			},
			expectError:   true,
			checkRedirect: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := buildHTTPClient(tc.config)

			// Check error expectation
			if tc.expectError && err == nil {
				t.Errorf("Expected error, but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			// If we expected error and got one, we're done
			if tc.expectError && err != nil {
				return
			}

			// Check client configuration
			// Check TLS configuration
			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("Expected *http.Transport, got %T", client.Transport)
			}
			if transport.TLSClientConfig.InsecureSkipVerify != tc.config.Insecure {
				t.Errorf("Expected InsecureSkipVerify to be %v, got %v",
					tc.config.Insecure, transport.TLSClientConfig.InsecureSkipVerify)
			}

			// Check redirect behavior
			if tc.checkRedirect {
				if client.CheckRedirect == nil {
					t.Errorf("Expected CheckRedirect to be set, but it's nil")
				}
			}
		})
	}
}

func TestExportToJSON(t *testing.T) {
	// Setup test data
	results := map[string]Result{
		"GET": {
			StatusCode: 200,
			Length:     100,
			Reason:     "OK",
		},
		"POST": {
			StatusCode: 201,
			Length:     150,
			Reason:     "Created",
		},
		"DELETE": {
			StatusCode: 405,
			Length:     50,
			Reason:     "Method Not Allowed",
		},
	}

	// Create temporary file
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "test-results.json")

	// Export to JSON
	err := exportToJSON(jsonFile, results)
	if err != nil {
		t.Fatalf("exportToJSON() returned error: %v", err)
	}

	// Read the file back
	content, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}

	// Check content
	for method, result := range results {
		expectedStatus := fmt.Sprintf("\"status_code\": %d", result.StatusCode)
		expectedLength := fmt.Sprintf("\"length\": %d", result.Length)
		expectedReason := fmt.Sprintf("\"reason\": \"%s\"", result.Reason)

		if !strings.Contains(string(content), expectedStatus) {
			t.Errorf("Expected JSON to contain %q", expectedStatus)
		}
		if !strings.Contains(string(content), expectedLength) {
			t.Errorf("Expected JSON to contain %q", expectedLength)
		}
		if !strings.Contains(string(content), expectedReason) {
			t.Errorf("Expected JSON to contain %q", expectedReason)
		}
		if !strings.Contains(string(content), fmt.Sprintf("\"%s\"", method)) {
			t.Errorf("Expected JSON to contain method %q", method)
		}
	}
}

func TestReadLinesFromFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-lines.txt")
	content := "line1\nline2\n# comment\n\nline3"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run the test
	lines, err := readLinesFromFile(testFile)
	if err != nil {
		t.Fatalf("readLinesFromFile(%q) returned error: %v", testFile, err)
	}

	// Verify results
	expected := []string{"line1", "line2", "line3"}
	if len(lines) != len(expected) {
		t.Errorf("Expected %d lines, got %d", len(expected), len(lines))
		t.Errorf("Got lines: %v", lines)
	}

	for i, expectedLine := range expected {
		if i >= len(lines) {
			t.Errorf("Missing expected line %d: %q", i, expectedLine)
			continue
		}
		if lines[i] != expectedLine {
			t.Errorf("Expected line %d to be %q, got %q", i, expectedLine, lines[i])
		}
	}
}

func TestIntegrationWithMockServer(t *testing.T) {
	// Create a mock server to test against
	allowedMethods := []string{"GET", "POST", "OPTIONS", "HEAD"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
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
	defer server.Close()

	// Setup test config
	config := Config{
		URL:     server.URL,
		Verbose: false,
		Quiet:   true,
		// Only test a few methods to speed up the test
		Wordlist: "", // No custom wordlist
		Threads:  2,
	}

	// Create a logger that won't output during tests
	logger := &Logger{
		Verbose: config.Verbose,
		Quiet:   config.Quiet,
	}

	// Test just a few methods
	methods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"}

	// Setup results storage
	results := make(map[string]Result)
	resultsMutex := &sync.Mutex{}

	// Build HTTP client
	client, err := buildHTTPClient(config)
	if err != nil {
		t.Fatalf("Failed to build HTTP client: %v", err)
	}

	// Test each method
	var wg sync.WaitGroup
	for _, method := range methods {
		wg.Add(1)
		go func(method string) {
			defer wg.Done()
			testMethod(client, config.URL, method, nil, nil, resultsMutex, results, logger)
		}(method)
	}

	wg.Wait()

	// Verify results
	for _, method := range methods {
		result, exists := results[method]
		if !exists {
			t.Errorf("No result found for method %q", method)
			continue
		}

		// Check if status code matches expectation
		expectedStatus := http.StatusMethodNotAllowed
		for _, m := range allowedMethods {
			if method == m {
				expectedStatus = http.StatusOK
				break
			}
		}

		if result.StatusCode != expectedStatus {
			t.Errorf("Method %q: expected status %d, got %d",
				method, expectedStatus, result.StatusCode)
		}
	}
}
