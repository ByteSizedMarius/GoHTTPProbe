package probe

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Config holds the configuration options for the HTTP probe
type Config struct {
	URL         string
	Verbose     bool
	Quiet       bool
	Insecure    bool
	FollowRedir bool
	SafeOnly    bool
	Wordlist    string
	Threads     int
	JSONFile    string
	Proxy       string
	Cookies     string
	Headers     []string
	InputFile   string
	CookieJar   string
	Timeout     int // in seconds
}

// Result represents the result of an HTTP method test
type Result struct {
	StatusCode int    `json:"status_code"`
	Length     int    `json:"length"`
	Reason     string `json:"reason"`
}

// DefaultMethods is the built-in list of HTTP methods to test
var DefaultMethods = []string{
	"CHECKIN", "CHECKOUT", "CONNECT", "COPY", "DELETE", "GET", "HEAD", "INDEX",
	"LINK", "LOCK", "MKCOL", "MOVE", "NOEXISTE", "OPTIONS", "ORDERPATCH",
	"PATCH", "POST", "PROPFIND", "PROPPATCH", "PUT", "REPORT", "SEARCH",
	"SHOWMETHOD", "SPACEJUMP", "TEXTSEARCH", "TRACE", "TRACK", "UNCHECKOUT",
	"UNLINK", "UNLOCK", "VERSION-CONTROL", "BAMBOOZLE",
}

// DangerousMethods lists methods that could be harmful to test
var DangerousMethods = map[string]bool{
	"DELETE":     true,
	"COPY":       true,
	"PUT":        true,
	"PATCH":      true,
	"UNCHECKOUT": true,
}

// Run executes the HTTP methods probe with the given configuration
func Run(config Config) error {
	logger := &Logger{
		Verbose: config.Verbose,
		Quiet:   config.Quiet,
	}
	logger.Info("Starting HTTP verb enumerating and tampering")

	// If input file specified, process multiple URLs
	if config.InputFile != "" {
		urls, err := readLinesFromFile(config.InputFile)
		if err != nil {
			return fmt.Errorf("failed to read URLs from file: %w", err)
		}

		for _, targetURL := range urls {
			if targetURL != "" {
				logger.Info("Testing URL: %s", targetURL)
				configCopy := config
				configCopy.URL = targetURL
				if err = runSingleProbe(configCopy, logger); err != nil {
					logger.Error("Error processing %s: %v", targetURL, err)
				}
			}
		}
		return nil
	}

	// Run probe on a single URL
	return runSingleProbe(config, logger)
}

// runSingleProbe runs the probe on a single URL
func runSingleProbe(config Config, logger *Logger) error {
	// Ensure URL has a protocol
	if config.URL != "" && !strings.Contains(config.URL, "://") {
		config.URL = "https://" + config.URL
		logger.Debug("Added https:// prefix to URL: %s", config.URL)
	}

	// Build HTTP client
	client, err := buildHTTPClient(config)
	if err != nil {
		return fmt.Errorf("failed to build HTTP client: %w", err)
	}

	// Parse headers
	headers, err := parseHeaders(config.Headers)
	if err != nil {
		return fmt.Errorf("failed to parse headers: %w", err)
	}

	// Parse cookies
	cookies, err := parseCookies(config.Cookies)
	if err != nil {
		return fmt.Errorf("failed to parse cookies: %w", err)
	}

	// Get methods to test
	methods, err := getMethods(config, logger)
	if err != nil {
		return fmt.Errorf("failed to get methods: %w", err)
	}

	// Filter out dangerous methods if safe mode is enabled
	if config.SafeOnly {
		var safeMethods []string
		for _, method := range methods {
			if !DangerousMethods[method] {
				safeMethods = append(safeMethods, method)
			}
		}
		methods = safeMethods
		logger.Info("Safe mode enabled, testing only non-dangerous methods")
	} else {
		// If not in safe mode, warn the user that dangerous methods will be tested
		logger.Warning("Testing includes potentially dangerous HTTP methods (PUT, DELETE, etc.)")
		logger.Warning("Use --safe-only to exclude them")
	}

	// Test the methods
	results := make(map[string]Result)
	var wg sync.WaitGroup
	resultsMutex := &sync.Mutex{}
	semaphore := make(chan struct{}, config.Threads)
	for _, method := range methods {
		wg.Add(1)
		go func(method string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			testMethod(client, config.URL, method, headers, cookies, resultsMutex, results, logger)
		}(method)
	}
	wg.Wait()

	// Sort results by method name
	sortedMethods := make([]string, 0, len(results))
	for method := range results {
		sortedMethods = append(sortedMethods, method)
	}
	sort.Strings(sortedMethods)

	// Print results
	if !config.Quiet {
		printResults(sortedMethods, results)
	}

	// Export to JSON if specified
	if config.JSONFile != "" {
		if err = exportToJSON(config.JSONFile, results); err != nil {
			return fmt.Errorf("failed to export results to JSON: %w", err)
		}
		logger.Success("Results exported to %s", config.JSONFile)
	}

	return nil
}

// buildHTTPClient creates an HTTP client based on the configuration
func buildHTTPClient(config Config) (*http.Client, error) {
	// Set default timeout to 10 seconds if not specified
	timeout := 10
	if config.Timeout > 0 {
		timeout = config.Timeout
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Configure TLS
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.Insecure},
	}

	// Configure proxy if specified
	if config.Proxy != "" {
		proxyURL, err := url.Parse(config.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	client.Transport = transport

	// Configure redirect behavior
	if !config.FollowRedir {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return client, nil
}

// parseHeaders processes header flags into a http.Header map
func parseHeaders(headerFlags []string) (http.Header, error) {
	headers := make(http.Header)

	for _, h := range headerFlags {
		// Check if it's a file
		if _, err := os.Stat(h); err == nil {
			var headerLines []string
			headerLines, err = readLinesFromFile(h)
			if err != nil {
				return nil, fmt.Errorf("failed to read headers from file: %w", err)
			}

			for _, line := range headerLines {
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				addHeader(headers, line)
			}
			continue
		}

		// Check if it's a comma-separated list
		if strings.Contains(h, ",") {
			for _, part := range strings.Split(h, ",") {
				addHeader(headers, strings.TrimSpace(part))
			}
			continue
		}

		// Single header
		addHeader(headers, h)
	}

	return headers, nil
}

// addHeader adds a single header to the header map
func addHeader(headers http.Header, headerLine string) {
	parts := strings.SplitN(headerLine, ":", 2)
	if len(parts) != 2 {
		return
	}
	headers.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
}

// parseCookies processes cookie string into a map
func parseCookies(cookieStr string) (map[string]string, error) {
	cookies := make(map[string]string)

	if cookieStr == "" {
		return cookies, nil
	}

	// Check if it's a file
	if _, err := os.Stat(cookieStr); err == nil {
		var cookieLines []string
		cookieLines, err = readLinesFromFile(cookieStr)
		if err != nil {
			return nil, fmt.Errorf("failed to read cookies from file: %w", err)
		}

		for _, line := range cookieLines {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			addCookie(cookies, line)
		}
		return cookies, nil
	}

	// Parse cookie string (format: "name1=value1; name2=value2")
	for _, part := range strings.Split(cookieStr, ";") {
		addCookie(cookies, strings.TrimSpace(part))
	}

	return cookies, nil
}

// addCookie adds a single cookie to the cookie map
func addCookie(cookies map[string]string, cookiePair string) {
	parts := strings.SplitN(cookiePair, "=", 2)
	if len(parts) != 2 {
		return
	}
	cookies[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
}

// getMethods retrieves the list of HTTP methods to test
func getMethods(config Config, logger *Logger) ([]string, error) {
	var methods []string

	// Start with default methods
	logger.Debug("Using %d default HTTP methods", len(DefaultMethods))
	methods = append(methods, DefaultMethods...)

	// Add methods from wordlist if specified
	if config.Wordlist != "" {
		wordlistMethods, err := readLinesFromFile(config.Wordlist)
		if err != nil {
			return nil, fmt.Errorf("failed to read wordlist: %w", err)
		}
		logger.Info("Added %d methods from wordlist: %s", len(wordlistMethods), config.Wordlist)
		methods = append(methods, wordlistMethods...)
	}

	// Try to get methods from OPTIONS request if server supports it
	logger.Debug("Sending OPTIONS request to discover supported methods")
	optionsMethods, err := getMethodsFromOptions(config, logger)
	if err != nil {
		logger.Warning("Failed to get methods from OPTIONS request: %v", err)
	} else if len(optionsMethods) > 0 {
		logger.Info("Added %d methods from OPTIONS response", len(optionsMethods))
		methods = append(methods, optionsMethods...)
	}

	// Normalize methods (uppercase and deduplicate)
	normalizedMethods := make([]string, 0, len(methods))
	seen := make(map[string]bool)
	for _, method := range methods {
		uppercase := strings.ToUpper(strings.TrimSpace(method))
		if uppercase != "" && !seen[uppercase] {
			normalizedMethods = append(normalizedMethods, uppercase)
			seen[uppercase] = true
		}
	}

	sort.Strings(normalizedMethods)
	return normalizedMethods, nil
}

// getMethodsFromOptions tries to get supported methods from an OPTIONS request
func getMethodsFromOptions(config Config, logger *Logger) ([]string, error) {
	// Skip if URL is empty
	if config.URL == "" {
		return nil, fmt.Errorf("no URL provided for OPTIONS request")
	}

	client, err := buildHTTPClient(config)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("OPTIONS", config.URL, nil)
	if err != nil {
		return nil, err
	}

	logger.Debug("Sending OPTIONS request with timeout of %d seconds", config.Timeout)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		logger.Debug("OPTIONS request returned status code %d", resp.StatusCode)
		return nil, err
	}

	allow := resp.Header.Get("Allow")
	if allow == "" {
		logger.Debug("OPTIONS request did not return Allow header")
		return nil, err
	}

	logger.Info("Server supports the following methods: %s", allow)
	var methods []string
	for _, method := range strings.Split(allow, ",") {
		methods = append(methods, strings.TrimSpace(method))
	}

	return methods, err
}

// testMethod tests a single HTTP method against the target URL
func testMethod(
	client *http.Client, targetURL, method string, headers http.Header,
	cookies map[string]string, mutex *sync.Mutex, results map[string]Result, logger *Logger,
) {
	req, err := http.NewRequest(method, targetURL, nil)
	if err != nil {
		logger.Debug("Failed to create request for method %s: %v", method, err)
		return
	}

	// Add headers
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Add cookies
	for name, value := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}

	logger.Debug("Testing method: %s", method)
	resp, err := client.Do(req)
	if err != nil {
		logger.Debug("Request failed for method %s: %v", method, err)
		mutex.Lock()
		results[method] = Result{
			StatusCode: 0,
			Length:     0,
			Reason:     err.Error(),
		}
		mutex.Unlock()
		return
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(resp.Body)

	// Read response body (limiting to avoid downloading large files)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Limit to 1MB
	if err != nil {
		logger.Debug("Failed to read response body for method %s: %v", method, err)
	}

	mutex.Lock()
	results[method] = Result{
		StatusCode: resp.StatusCode,
		Length:     len(body),
		Reason:     resp.Status,
	}
	mutex.Unlock()
}

// printResults prints the test results in a table format
func printResults(methods []string, results map[string]Result) {
	// Print header
	fmt.Printf("\n%-15s %-10s %-10s %s\n", "METHOD", "STATUS", "LENGTH", "REASON")
	fmt.Printf("%-15s %-10s %-10s %s\n", "------", "------", "------", "------")

	// Print results
	for _, method := range methods {
		result := results[method]
		var statusColor, reasonColor string

		// Determine color based on status code
		switch {
		case result.StatusCode == 200:
			statusColor = "\033[32m" // Green
			reasonColor = "\033[32m"
		case result.StatusCode >= 300 && result.StatusCode < 400:
			statusColor = "\033[36m" // Cyan
			reasonColor = "\033[36m"
		case result.StatusCode >= 400 && result.StatusCode < 500:
			statusColor = "\033[31m" // Red
			reasonColor = "\033[31m"
		case result.StatusCode >= 500 && result.StatusCode != 502:
			statusColor = "\033[33m" // Yellow
			reasonColor = "\033[33m"
		case result.StatusCode == 502:
			statusColor = "\033[33m" // Yellow
			reasonColor = "\033[33m"
		default:
			statusColor = ""
			reasonColor = ""
		}

		resetColor := "\033[0m"
		fmt.Printf("%-15s %s%-10d%s %-10d %s%s%s\n",
			method,
			statusColor, result.StatusCode, resetColor,
			result.Length,
			reasonColor, result.Reason, resetColor,
		)
	}
	fmt.Println()
}

// exportToJSON exports results to a JSON file
func exportToJSON(filename string, results map[string]Result) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results to JSON: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// readLinesFromFile reads lines from a file
func readLinesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
