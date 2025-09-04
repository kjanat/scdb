package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Config holds the downloader configuration
type Config struct {
	Username       string
	Password       string
	OutputDir      string
	Countries      []string
	DisplayType    int // 1=Split all, 2=Split speed/red, 3=All in one, 4=All in one (alt icon)
	DangerZones    bool
	IconSize       int // 1=22x22, 2=24x24, 3=32x32, 4=48x48, 5=80x80
	DownloadFixed  bool
	DownloadMobile bool
	Verbose        bool
}

// SCDBDownloader handles the download process
type SCDBDownloader struct {
	client *http.Client
	config *Config
}

// NewDownloader creates a new SCDB downloader instance
func NewDownloader(cfg *Config) *SCDBDownloader {
	jar, _ := cookiejar.New(nil)

	client := &http.Client{
		Timeout: time.Minute * 5,
		Jar:     jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // For self-signed certificates
			},
		},
	}

	return &SCDBDownloader{
		client: client,
		config: cfg,
	}
}

// login authenticates with the SCDB website
func (d *SCDBDownloader) login() error {
	if d.config.Verbose {
		fmt.Println("Logging in to SCDB...")
	}

	// First GET the login page to extract the CSRF token
	resp, err := d.client.Get("https://www.scdb.info/en/login/")
	if err != nil {
		return fmt.Errorf("failed to get login page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login page: %w", err)
	}

	// Extract the dynamic CSRF token from the form
	tokenPattern := regexp.MustCompile(`name="([a-f0-9]{40})" value="([a-f0-9]{40})"`)
	matches := tokenPattern.FindStringSubmatch(string(body))
	if len(matches) < 3 {
		return fmt.Errorf("failed to find CSRF token in login page")
	}

	tokenName := matches[1]
	tokenValue := matches[2]

	if d.config.Verbose {
		fmt.Printf("Found CSRF token: %s=%s\n", tokenName, tokenValue)
	}

	// Prepare login form data with dynamic token
	formData := url.Values{
		tokenName:      {tokenValue},
		"u_name":       {d.config.Username},
		"u_password":   {d.config.Password},
		"login_submit": {"Login"},
	}

	req, err := http.NewRequest("POST", "https://www.scdb.info/en/login/",
		bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	req.Header.Set("Origin", "https://www.scdb.info")
	req.Header.Set("Referer", "https://www.scdb.info/en/login/")

	resp, err = d.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check if login was successful by following redirects
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	if d.config.Verbose {
		fmt.Println("Login successful!")
	}

	return nil
}

// downloadFixed downloads the fixed speed camera database
func (d *SCDBDownloader) downloadFixed() error {
	if d.config.Verbose {
		fmt.Println("Downloading fixed speed cameras...")
	}

	// Build country selection
	formData := url.Values{
		"download_agreement_accept":         {"1"},
		"download_wave_right_of_rescission": {"1"},
		"typ":                               {fmt.Sprintf("%d", d.config.DisplayType)},
		"dangerzones":                       {"1"}, // Default to enabled, will be overridden below
		"vorwarnzeit":                       {""},  // Empty warning time field as in PowerShell
		"iconsize":                          {fmt.Sprintf("%d", d.config.IconSize)},
		"download_start":                    {"Download+Now"},
	}

	// Add danger zones setting
	if d.config.DangerZones {
		formData.Set("dangerzones", "1")
	} else {
		formData.Set("dangerzones", "0")
	}

	// Add countries
	for _, country := range d.config.Countries {
		formData.Add("land[]", country)
	}

	req, err := http.NewRequest("POST", "https://www.scdb.info/my/downloadsection",
		bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Origin", "https://www.scdb.info")
	req.Header.Set("Referer", "https://www.scdb.info/my/downloadsection")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Save to file
	outputPath := filepath.Join(d.config.OutputDir, "garmin.zip")
	return d.saveResponseToFile(resp, outputPath)
}

// downloadMobile downloads the mobile speed camera database
func (d *SCDBDownloader) downloadMobile() error {
	if d.config.Verbose {
		fmt.Println("Downloading mobile speed cameras...")
	}

	formData := url.Values{
		"mobile_submit": {"Download+For+Free"},
	}

	req, err := http.NewRequest("POST", "https://www.scdb.info/intern/download/garmin-mobile.zip",
		bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create mobile download request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Origin", "https://www.scdb.info")
	req.Header.Set("Referer", "https://www.scdb.info/my/")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("mobile download request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Save to file
	outputPath := filepath.Join(d.config.OutputDir, "garmin-mobile.zip")
	return d.saveResponseToFile(resp, outputPath)
}

// saveResponseToFile saves the HTTP response body to a file
func (d *SCDBDownloader) saveResponseToFile(resp *http.Response, filepath string) error {
	// Check content type and response
	contentType := resp.Header.Get("Content-Type")
	if d.config.Verbose {
		fmt.Printf("Response status: %d, Content-Type: %s\n", resp.StatusCode, contentType)
	}

	if !strings.Contains(contentType, "zip") && !strings.Contains(contentType, "octet") {
		// Read response body for error message
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected response (not a zip file), Content-Type: %s, Body: %s", contentType, string(body))
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = out.Close() }()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	if d.config.Verbose {
		fmt.Printf("Downloaded %d bytes to %s\n", written, filepath)
	}

	return nil
}

// Run executes the download process
func (d *SCDBDownloader) Run() error {
	// Login first
	if err := d.login(); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Download fixed cameras if requested
	if d.config.DownloadFixed {
		if err := d.downloadFixed(); err != nil {
			return fmt.Errorf("failed to download fixed cameras: %w", err)
		}
	}

	// Download mobile cameras if requested
	if d.config.DownloadMobile {
		if err := d.downloadMobile(); err != nil {
			return fmt.Errorf("failed to download mobile cameras: %w", err)
		}
	}

	return nil
}

// getAllCountries returns all available country codes
func getAllCountries() []string {
	return []string{
		"AFG", "DZ", "AND", "RA", "ARM", "AUS", "A", "AZ", "BRN", "BY", "B", "BZ", "BIH",
		"BR", "BG", "CDN", "RCH", "CO", "HR", "CY", "CZ", "DK", "EC", "ET", "ES2", "EST",
		"FJI", "FI", "FR", "GF", "GE", "D", "GBZ", "GR", "GP", "GT", "GUY", "HN", "HK",
		"H", "IS", "IND", "IR", "IRQ", "IRL", "IL", "I", "J", "JOR", "KZ", "KWT", "KS",
		"LAO", "LV", "RL", "LI", "LT", "L", "MO", "MAL", "M", "MQ", "MS", "MEX", "MD",
		"MGL", "MA", "NAM", "NL", "NZ", "MK", "NO", "OM", "PK", "PA", "PY", "PE", "RP",
		"PL", "P", "Q", "RO", "RUS", "RWA", "RE", "RSM", "KSA", "SRB", "SGP", "SK", "SLO",
		"ZA", "ROK", "ES", "SE", "CH", "RCT", "T", "TT", "TN", "TR", "UA", "UAE", "GB",
		"USA", "ROU", "UZ", "VN", "Z", "ZW",
	}
}

func main() {
	var config Config

	// Parse command line flags
	flag.StringVar(&config.Username, "user", "", "SCDB username (required, or use SCDB_USER env var)")
	flag.StringVar(&config.Password, "pass", "", "SCDB password (required, or use SCDB_PASS env var)")
	flag.StringVar(&config.OutputDir, "output", ".", "Output directory for downloads")

	countries := flag.String("countries", "all", "Comma-separated country codes or 'all' for all countries")
	flag.IntVar(&config.DisplayType, "display", 1, "Display type (1=Split all, 2=Split speed/red, 3=All in one, 4=Alt icon)")
	flag.BoolVar(&config.DangerZones, "dangerzones", true, "Include danger zones")
	flag.IntVar(&config.IconSize, "iconsize", 5, "Icon size (1=22x22, 2=24x24, 3=32x32, 4=48x48, 5=80x80)")

	flag.BoolVar(&config.DownloadFixed, "fixed", true, "Download fixed speed cameras")
	flag.BoolVar(&config.DownloadMobile, "mobile", true, "Download mobile speed cameras")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")

	flag.Parse()

	// Use environment variables if flags not provided
	if config.Username == "" {
		config.Username = os.Getenv("SCDB_USER")
	}
	if config.Password == "" {
		config.Password = os.Getenv("SCDB_PASS")
	}

	// Validate required fields
	if config.Username == "" || config.Password == "" {
		fmt.Fprintf(os.Stderr, "Error: username and password are required\n")
		fmt.Fprintf(os.Stderr, "Provide via -user/-pass flags or SCDB_USER/SCDB_PASS environment variables\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Parse countries
	if *countries == "all" {
		config.Countries = getAllCountries()
	} else {
		config.Countries = strings.Split(*countries, ",")
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Create downloader and run
	downloader := NewDownloader(&config)
	if err := downloader.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		os.Exit(1)
	}

	if config.Verbose {
		fmt.Println("Downloads completed successfully!")
	}
}
