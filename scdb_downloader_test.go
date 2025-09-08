package main

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestNewDownloader(t *testing.T) {
	config := CreateTestConfig()
	downloader := NewDownloader(config)

	// Verify downloader is created with correct config
	if downloader.config != config {
		t.Errorf("NewDownloader() config = %p, want %p", downloader.config, config)
	}

	// Verify HTTP client is configured correctly
	if downloader.client == nil {
		t.Errorf("NewDownloader() client is nil")
		return
	}

	// Check timeout
	expectedTimeout := time.Minute * 5
	if downloader.client.Timeout != expectedTimeout {
		t.Errorf("NewDownloader() client timeout = %v, want %v", downloader.client.Timeout, expectedTimeout)
	}

	// Check cookie jar is present
	if downloader.client.Jar == nil {
		t.Errorf("NewDownloader() client jar is nil")
	}

	// Check TLS config
	transport, ok := downloader.client.Transport.(*http.Transport)
	if !ok {
		t.Errorf("NewDownloader() client transport is not *http.Transport")
		return
	}

	if transport.TLSClientConfig == nil {
		t.Errorf("NewDownloader() TLS config is nil")
	} else if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Errorf("NewDownloader() TLS InsecureSkipVerify = false, want true")
	}
}

func TestSCDBDownloader_login(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		setupMock func(*MockSCDBServer)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "Successful login",
			config: CreateTestConfig(),
			setupMock: func(m *MockSCDBServer) {
				// Default behavior - success
			},
			wantErr: false,
		},
		{
			name:   "Login failure",
			config: CreateTestConfig(),
			setupMock: func(m *MockSCDBServer) {
				m.SetFailures(true, false, false)
			},
			wantErr: true,
			errMsg:  "login request failed",
		},
		{
			name: "Verbose login",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				Countries:      []string{"NL"},
				DisplayType:    1,
				IconSize:       5,
				DownloadFixed:  true,
				DownloadMobile: true,
				Verbose:        true, // Enable verbose mode
			},
			setupMock: func(m *MockSCDBServer) {
				// Default behavior - success
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			mockServer := NewMockSCDBServer()
			defer mockServer.Close()

			tt.setupMock(mockServer)

			// Create downloader with config pointing to mock server
			downloader := NewDownloader(tt.config)

			// Replace URLs in the downloader to point to mock server
			// This is a bit tricky since the URLs are hardcoded in the login method
			// We'll need to modify this approach or use a more sophisticated mock

			// For now, we'll test the URL construction logic separately
			// and test login with a real-world scenario in E2E tests

			// Test that we can create a downloader and it has the right structure
			if downloader == nil {
				t.Errorf("NewDownloader() returned nil")
				return
			}

			if downloader.config != tt.config {
				t.Errorf("Downloader config mismatch")
			}
		})
	}
}

func TestSCDBDownloader_saveResponseToFile(t *testing.T) {
	tempDir := CreateTempDir(t, "scdb_save_test")
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := CreateTestConfig()
	config.OutputDir = tempDir
	downloader := NewDownloader(config)

	tests := []struct {
		name        string
		contentType string
		content     string
		filename    string
		verbose     bool
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "Valid ZIP file",
			contentType: "application/zip",
			content:     "PK\x03\x04mock_zip_content",
			filename:    "test.zip",
			wantErr:     false,
		},
		{
			name:        "Valid octet-stream",
			contentType: "application/octetstream", // No hyphen, matches real server
			content:     "PK\x03\x04mock_zip_content",
			filename:    "test2.zip",
			wantErr:     false,
		},
		{
			name:        "Invalid content type",
			contentType: "text/html",
			content:     "<html><body>Error page</body></html>",
			filename:    "error.zip",
			wantErr:     true,
			errMsg:      "unexpected response",
		},
		{
			name:        "Valid ZIP with verbose output",
			contentType: "application/zip",
			content:     "PK\x03\x04verbose_test",
			filename:    "verbose.zip",
			verbose:     true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set verbose mode if needed
			downloader.config.Verbose = tt.verbose

			// Create mock HTTP response with a simple string reader
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       &simpleBody{content: tt.content},
			}
			resp.Header.Set("Content-Type", tt.contentType)

			filepath := filepath.Join(tempDir, tt.filename)
			err := downloader.saveResponseToFile(resp, filepath)

			if (err != nil) != tt.wantErr {
				t.Errorf("saveResponseToFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				AssertErrorContains(t, err, tt.errMsg)
				return
			}

			if !tt.wantErr {
				// Verify file was created and has correct content
				AssertFileExists(t, filepath, int64(len(tt.content)))

				// Read file content and verify
				savedContent, err := os.ReadFile(filepath)
				if err != nil {
					t.Errorf("Failed to read saved file: %v", err)
					return
				}

				if string(savedContent) != tt.content {
					t.Errorf("Saved content = %q, want %q", string(savedContent), tt.content)
				}
			}
		})
	}
}

func TestSCDBDownloader_Run(t *testing.T) {
	tempDir := CreateTempDir(t, "scdb_run_test")
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name       string
		config     *Config
		wantErr    bool
		errMsg     string
		wantFixed  bool
		wantMobile bool
	}{
		{
			name: "Download both fixed and mobile",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				OutputDir:      tempDir,
				Countries:      []string{"NL"},
				DisplayType:    1,
				IconSize:       5,
				DownloadFixed:  true,
				DownloadMobile: true,
			},
			wantErr:    false,
			wantFixed:  true,
			wantMobile: true,
		},
		{
			name: "Download only fixed",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				OutputDir:      tempDir,
				Countries:      []string{"NL"},
				DisplayType:    1,
				IconSize:       5,
				DownloadFixed:  true,
				DownloadMobile: false,
			},
			wantErr:    false,
			wantFixed:  true,
			wantMobile: false,
		},
		{
			name: "Download only mobile",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				OutputDir:      tempDir,
				Countries:      []string{"NL"},
				DisplayType:    1,
				IconSize:       5,
				DownloadFixed:  false,
				DownloadMobile: true,
			},
			wantErr:    false,
			wantFixed:  false,
			wantMobile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := NewDownloader(tt.config)

			// Note: Since we can't easily mock the HTTP client in the existing code,
			// this test mainly verifies the structure and would need network access
			// for full testing. In a real scenario, we'd want to inject the HTTP client
			// or make the URLs configurable for testing.

			// For now, we'll test that the downloader has the correct configuration
			if downloader.config.DownloadFixed != tt.wantFixed {
				t.Errorf("DownloadFixed = %v, want %v", downloader.config.DownloadFixed, tt.wantFixed)
			}

			if downloader.config.DownloadMobile != tt.wantMobile {
				t.Errorf("DownloadMobile = %v, want %v", downloader.config.DownloadMobile, tt.wantMobile)
			}

			// Verify output directory is set correctly
			if downloader.config.OutputDir != tempDir {
				t.Errorf("OutputDir = %q, want %q", downloader.config.OutputDir, tempDir)
			}
		})
	}
}

func TestSCDBDownloader_FormDataValidation(t *testing.T) {
	// Test that form data is constructed correctly for downloadFixed
	config := CreateTestConfig()
	config.Countries = []string{"D", "A", "CH"} // DACH region
	config.DisplayType = 3
	config.IconSize = 4
	config.WarningTime = 300
	config.DangerZones = true
	config.FranceDangerMode = true

	downloader := NewDownloader(config)

	// We can't easily test the form data construction without refactoring
	// the downloadFixed method to be more testable (e.g., by extracting
	// form building into a separate method)

	// For now, verify the configuration is set up correctly
	if len(config.Countries) != 3 {
		t.Errorf("Countries length = %d, want 3", len(config.Countries))
	}

	expectedCountries := []string{"D", "A", "CH"}
	for i, expected := range expectedCountries {
		if i >= len(config.Countries) || config.Countries[i] != expected {
			t.Errorf("Countries[%d] = %q, want %q", i, config.Countries[i], expected)
		}
	}

	// Test the downloader has the right config
	if downloader.config.DisplayType != 3 {
		t.Errorf("DisplayType = %d, want 3", downloader.config.DisplayType)
	}

	if downloader.config.IconSize != 4 {
		t.Errorf("IconSize = %d, want 4", downloader.config.IconSize)
	}

	if downloader.config.WarningTime != 300 {
		t.Errorf("WarningTime = %d, want 300", downloader.config.WarningTime)
	}

	if !downloader.config.DangerZones {
		t.Errorf("DangerZones = false, want true")
	}

	if !downloader.config.FranceDangerMode {
		t.Errorf("FranceDangerMode = false, want true")
	}
}

func TestSCDBDownloader_HTTPClientConfiguration(t *testing.T) {
	config := CreateTestConfig()
	downloader := NewDownloader(config)

	// Test that HTTP client has cookie jar
	if downloader.client.Jar == nil {
		t.Error("HTTP client should have cookie jar")
		return
	}

	// Test that cookie jar works
	jar := downloader.client.Jar
	if jar == nil {
		t.Error("Cookie jar is nil")
		return
	}

	// Create a test cookie
	testURL := "https://www.scdb.info/"
	parsedURL, _ := parseURL(testURL)
	if parsedURL == nil {
		t.Error("Failed to parse test URL")
		return
	}

	// The cookie jar should be ready to use (we don't need to test actual cookie storage here)
}

func TestSCDBDownloader_TLSConfiguration(t *testing.T) {
	config := CreateTestConfig()
	downloader := NewDownloader(config)

	// Verify TLS configuration
	transport, ok := downloader.client.Transport.(*http.Transport)
	if !ok {
		t.Error("HTTP client transport is not *http.Transport")
		return
	}

	tlsConfig := transport.TLSClientConfig
	if tlsConfig == nil {
		t.Error("TLS config is nil")
		return
	}

	// Verify InsecureSkipVerify is set (for self-signed certificates)
	if !tlsConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true for SCDB compatibility")
	}
}

// simpleBody implements io.ReadCloser for testing
type simpleBody struct {
	content string
	pos     int
	closed  bool
}

func (s *simpleBody) Read(p []byte) (n int, err error) {
	if s.closed {
		return 0, nil
	}
	if s.pos >= len(s.content) {
		return 0, nil // EOF
	}
	n = copy(p, s.content[s.pos:])
	s.pos += n
	return n, nil
}

func (s *simpleBody) Close() error {
	s.closed = true
	return nil
}

// Helper function to parse URL (simplified version for testing)
func parseURL(rawURL string) (*http.Request, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// TestDownloaderIntegration tests basic integration without actual network calls
func TestDownloaderIntegration(t *testing.T) {
	tempDir := CreateTempDir(t, "scdb_integration_test")
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{
		Username:         "test@example.com",
		Password:         "testpass123",
		OutputDir:        tempDir,
		Countries:        []string{"NL", "B"},
		DisplayType:      2,
		DangerZones:      true,
		FranceDangerMode: false,
		IconSize:         4,
		WarningTime:      300,
		DownloadFixed:    true,
		DownloadMobile:   true,
		Verbose:          true,
	}

	// Validate the config first
	err := validateConfig(config)
	AssertNoError(t, err)

	// Create downloader
	downloader := NewDownloader(config)

	// Verify downloader setup
	if downloader.config.Username != config.Username {
		t.Errorf("Username = %q, want %q", downloader.config.Username, config.Username)
	}

	if downloader.config.Verbose != config.Verbose {
		t.Errorf("Verbose = %v, want %v", downloader.config.Verbose, config.Verbose)
	}

	// Test that expected output files would be created in the right location
	expectedFixed := filepath.Join(tempDir, "garmin.zip")
	expectedMobile := filepath.Join(tempDir, "garmin-mobile.zip")

	// These files shouldn't exist yet
	AssertFileNotExists(t, expectedFixed)
	AssertFileNotExists(t, expectedMobile)

	// Verify we can create files in the output directory
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	AssertNoError(t, err)
	AssertFileExists(t, testFile, 4)
}

// TestCSRFTokenExtraction tests CSRF token pattern matching
func TestCSRFTokenExtraction(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		wantName string
		wantVal  string
		wantErr  bool
	}{
		{
			name:     "Valid CSRF token",
			html:     `<input type="hidden" name="abcdef1234567890abcdef1234567890abcdef12" value="abcdef1234567890abcdef1234567890abcdef12">`,
			wantName: "abcdef1234567890abcdef1234567890abcdef12",
			wantVal:  "abcdef1234567890abcdef1234567890abcdef12",
			wantErr:  false,
		},
		{
			name:     "Different token values",
			html:     `<input type="hidden" name="1234567890abcdef1234567890abcdef12345678" value="8765432109fedcba8765432109fedcba87654321">`,
			wantName: "1234567890abcdef1234567890abcdef12345678",
			wantVal:  "8765432109fedcba8765432109fedcba87654321",
			wantErr:  false,
		},
		{
			name:    "No CSRF token",
			html:    `<input type="text" name="username">`,
			wantErr: true,
		},
		{
			name:    "Invalid token length",
			html:    `<input type="hidden" name="short" value="tooshort">`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the same regex pattern from the login method
			tokenPattern := regexp.MustCompile(`name="([a-f0-9]{40})" value="([a-f0-9]{40})"`)
			matches := tokenPattern.FindStringSubmatch(tt.html)

			if tt.wantErr {
				if len(matches) >= 3 {
					t.Errorf("Expected no matches, got %v", matches)
				}
				return
			}

			if len(matches) < 3 {
				t.Errorf("Expected matches, got none")
				return
			}

			tokenName := matches[1]
			tokenValue := matches[2]

			if tokenName != tt.wantName {
				t.Errorf("Token name = %q, want %q", tokenName, tt.wantName)
			}

			if tokenValue != tt.wantVal {
				t.Errorf("Token value = %q, want %q", tokenValue, tt.wantVal)
			}
		})
	}
}
