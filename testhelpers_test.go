package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// MockSCDBServer creates a mock server that simulates SCDB responses
type MockSCDBServer struct {
	server      *httptest.Server
	loginCalls  int
	fixedCalls  int
	mobileCalls int
	failLogin   bool
	failFixed   bool
	failMobile  bool
	csrfToken   string
}

// NewMockSCDBServer creates a new mock server for testing
func NewMockSCDBServer() *MockSCDBServer {
	mock := &MockSCDBServer{
		csrfToken: "abcdef1234567890abcdef1234567890abcdef12", // 40 char hex string
	}

	mux := http.NewServeMux()

	// Login page - handles both GET and POST
	mux.HandleFunc("/en/login/", mock.handleLogin)

	// Fixed cameras download
	mux.HandleFunc("/my/downloadsection", mock.handleFixedDownload)

	// Mobile cameras download
	mux.HandleFunc("/intern/download/garmin-mobile.zip", mock.handleMobileDownload)

	mock.server = httptest.NewServer(mux)

	// Add timeout controls to prevent test hangs
	mock.server.Config.ReadTimeout = 10 * time.Second
	mock.server.Config.WriteTimeout = 10 * time.Second
	mock.server.Config.IdleTimeout = 10 * time.Second

	return mock
}

// Close shuts down the mock server
func (m *MockSCDBServer) Close() {
	m.server.Close()
}

// URL returns the base URL of the mock server
func (m *MockSCDBServer) URL() string {
	return m.server.URL
}

// SetFailures configures the mock server to simulate failures
func (m *MockSCDBServer) SetFailures(login, fixed, mobile bool) {
	m.failLogin = login
	m.failFixed = fixed
	m.failMobile = mobile
}

// GetStats returns call statistics
func (m *MockSCDBServer) GetStats() (login, fixed, mobile int) {
	return m.loginCalls, m.fixedCalls, m.mobileCalls
}

// handleLogin processes both GET (login page) and POST (login attempt)
func (m *MockSCDBServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Serve login page with CSRF token
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><title>SCDB Login</title></head>
<body>
<form method="POST" action="/en/login/">
	<input type="hidden" name="%s" value="%s">
	<input type="text" name="u_name" placeholder="Username">
	<input type="password" name="u_password" placeholder="Password">
	<input type="submit" name="login_submit" value="Login">
</form>
</body>
</html>
`, m.csrfToken, m.csrfToken)

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.loginCalls++

	if m.failLogin {
		http.Error(w, "Login failed", http.StatusUnauthorized)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Check CSRF token
	tokenValue := r.FormValue(m.csrfToken)
	if tokenValue != m.csrfToken {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	// Check credentials
	username := r.FormValue("u_name")
	password := r.FormValue("u_password")

	if username == "" || password == "" {
		http.Error(w, "Missing credentials", http.StatusBadRequest)
		return
	}

	// Simulate successful login with redirect
	w.Header().Set("Set-Cookie", "PHPSESSID=test_session_id; Path=/")
	w.Header().Set("Location", "/my/")
	w.WriteHeader(http.StatusFound)
}

// handleFixedDownload processes fixed camera download requests
func (m *MockSCDBServer) handleFixedDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.fixedCalls++

	if m.failFixed {
		http.Error(w, "Download failed", http.StatusInternalServerError)
		return
	}

	// Parse form to validate required fields
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Check required form fields
	requiredFields := []string{"download_agreement_accept", "download_wave_right_of_rescission", "typ", "iconsize", "download_start"}
	for _, field := range requiredFields {
		if r.FormValue(field) == "" {
			http.Error(w, fmt.Sprintf("Missing required field: %s", field), http.StatusBadRequest)
			return
		}
	}

	// Check that countries are specified
	countries := r.Form["land[]"]
	if len(countries) == 0 {
		http.Error(w, "No countries specified", http.StatusBadRequest)
		return
	}

	// Return mock ZIP content
	mockZipContent := "PK\x03\x04mock_garmin_zip_content_here"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=garmin.zip")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(mockZipContent))
}

// handleMobileDownload processes mobile camera download requests
func (m *MockSCDBServer) handleMobileDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mobileCalls++

	if m.failMobile {
		http.Error(w, "Download failed", http.StatusInternalServerError)
		return
	}

	// Return mock ZIP content
	mockZipContent := "PK\x03\x04mock_mobile_zip_content_here"
	w.Header().Set("Content-Type", "application/octetstream") // Note: no hyphen, matches real server
	w.Header().Set("Content-Disposition", "attachment; filename=garmin-mobile.zip")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(mockZipContent))
}

// CreateTestConfig creates a test configuration with reasonable defaults
func CreateTestConfig() *Config {
	return &Config{
		Username:         "testuser",
		Password:         "testpass",
		OutputDir:        ".",
		Countries:        []string{"NL", "B"},
		DisplayType:      2,
		DangerZones:      true,
		FranceDangerMode: false,
		IconSize:         4,
		WarningTime:      300,
		DownloadFixed:    true,
		DownloadMobile:   true,
		Verbose:          false,
	}
}

// CreateTestDownloader creates a downloader instance for testing
func CreateTestDownloader(config *Config) *SCDBDownloader {
	return NewDownloader(config)
}

// CreateTempDir creates a temporary directory for testing
func CreateTempDir(t *testing.T, prefix string) string {
	tempDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tempDir
}

// AssertFileExists checks that a file exists and optionally validates its size
func AssertFileExists(t *testing.T, filepath string, minSize int64) {
	t.Helper()

	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist, but it doesn't", filepath)
		return
	}
	if err != nil {
		t.Errorf("Error checking file %s: %v", filepath, err)
		return
	}

	if minSize > 0 && info.Size() < minSize {
		t.Errorf("File %s size %d bytes, expected at least %d bytes", filepath, info.Size(), minSize)
	}
}

// AssertFileNotExists checks that a file does not exist
func AssertFileNotExists(t *testing.T, filepath string) {
	t.Helper()

	if _, err := os.Stat(filepath); !os.IsNotExist(err) {
		t.Errorf("Expected file %s to not exist, but it does", filepath)
	}
}

// AssertErrorContains checks that an error contains a specific substring
func AssertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()

	if err == nil {
		t.Errorf("Expected error containing %q, got nil", substr)
		return
	}

	if !strings.Contains(err.Error(), substr) {
		t.Errorf("Expected error to contain %q, got %q", substr, err.Error())
	}
}

// AssertNoError checks that no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// MockHTMLResponse creates an HTML response with CSRF token for testing
func MockHTMLResponse(csrfToken string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
<form>
	<input type="hidden" name="%s" value="%s">
	<input type="submit" value="Submit">
</form>
</body>
</html>
`, csrfToken, csrfToken)
}

// MockErrorHTMLResponse creates an HTML error response without CSRF token
func MockErrorHTMLResponse() string {
	return `
<!DOCTYPE html>
<html>
<head><title>Error</title></head>
<body>
<h1>Error: No CSRF token found</h1>
</body>
</html>
`
}

// MockInvalidHTMLResponse creates malformed HTML for testing error handling
func MockInvalidHTMLResponse() string {
	return `<html><head><title>Invalid</title></head><body><p>No form or token</p></body></html>`
}

// ValidateFormData checks that HTTP form data contains expected fields
func ValidateFormData(t *testing.T, r *http.Request, expectedFields map[string]string) {
	t.Helper()

	err := r.ParseForm()
	if err != nil {
		t.Errorf("Failed to parse form data: %v", err)
		return
	}

	for field, expectedValue := range expectedFields {
		actualValue := r.FormValue(field)
		if actualValue != expectedValue {
			t.Errorf("Form field %s = %q, want %q", field, actualValue, expectedValue)
		}
	}
}

// CountriesTestData provides test data for country-related tests
func CountriesTestData() map[string][]string {
	return map[string][]string{
		"dach":        {"D", "A", "CH"},
		"benelux":     {"B", "NL", "L"},
		"scandinavia": {"SE", "NO", "DK", "FI", "IS"},
		"single":      {"NL"},
		"multiple":    {"NL", "B", "D", "FR"},
	}
}
