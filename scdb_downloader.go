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

	yaml "gopkg.in/yaml.v2"
)

// Config holds the downloader configuration
type Config struct {
	Username         string   `yaml:"username"`
	Password         string   `yaml:"password"`
	OutputDir        string   `yaml:"output_dir"`
	Countries        []string `yaml:"countries"`
	DisplayType      int      `yaml:"display_type"`       // 1=Split all, 2=Split speed/red, 3=All in one, 4=All in one (alt icon)
	DangerZones      bool     `yaml:"danger_zones"`       // Include danger zones
	FranceDangerMode bool     `yaml:"france_danger_mode"` // true=Display as danger zone, false=Display correct position
	IconSize         int      `yaml:"icon_size"`          // 1=22x22, 2=24x24, 3=32x32, 4=48x48, 5=80x80
	WarningTime      int      `yaml:"warning_time"`       // Warning time in seconds (0 = disabled, default)
	DownloadFixed    bool     `yaml:"download_fixed"`     // Download fixed speed cameras
	DownloadMobile   bool     `yaml:"download_mobile"`    // Download mobile speed cameras
	Verbose          bool     `yaml:"verbose"`            // Enable verbose output
	ConfigFile       string   `yaml:"-"`                  // Config file path (not saved in config)
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

	// First, GET the login page to extract the CSRF token
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

	// Prepare login form data with a dynamic token
	formData := url.Values{
		tokenName:      []string{tokenValue},
		"u_name":       []string{d.config.Username},
		"u_password":   []string{d.config.Password},
		"login_submit": []string{"Login"},
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
		"vorwarnzeit":                       {fmt.Sprintf("%d", d.config.WarningTime)},
		"iconsize":                          {fmt.Sprintf("%d", d.config.IconSize)},
		"download_start":                    {"Download+Now"},
	}

	// Add France-specific danger zone handling
	if d.config.FranceDangerMode {
		formData.Set("france_danger", "1") // Display position as a danger zone
	} else {
		formData.Set("france_danger", "0") // Display the correct position
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
		// Read the response body for an error message
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

// Country and region mappings
var (
	allCountries = []string{
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

	// Regional presets based on the web interface
	regionMap = map[string][]string{
		"africa":       {"AFG", "DZ", "ET", "MA", "NAM", "ZA", "RWA", "TN", "Z", "ZW"},
		"asia":         {"ARM", "AZ", "BRN", "HK", "IND", "IR", "IRQ", "IL", "J", "JOR", "KZ", "KWT", "KS", "LAO", "MAL", "MO", "MGL", "OM", "PK", "RP", "SGP", "ROK", "RCT", "T", "UAE", "UZ", "VN"},
		"europe":       {"AND", "A", "BY", "B", "BIH", "BG", "HR", "CY", "CZ", "DK", "EST", "FI", "FR", "GE", "D", "GBZ", "GR", "H", "IS", "IRL", "I", "LV", "RL", "LI", "LT", "L", "M", "MK", "NO", "PL", "P", "RO", "RUS", "RSM", "SRB", "SK", "SLO", "ES", "SE", "CH", "TR", "UA", "GB"},
		"northamerica": {"CDN", "USA", "MEX", "GT", "HN", "BZ", "PA", "TT"},
		"southamerica": {"RA", "BR", "RCH", "CO", "EC", "GUY", "PY", "PE", "ROU"},
		"oceania":      {"AUS", "FJI", "NZ"},
		"dach":         {"D", "A", "CH"}, // Germany/Austria/Switzerland
		"benelux":      {"B", "NL", "L"}, // Belgium/Netherlands/Luxembourg
		"westeurope":   {"B", "NL", "L", "FR", "D", "A", "CH", "I", "ES", "P", "GB", "IRL"},
		"easteurope":   {"PL", "CZ", "SK", "H", "RO", "BG", "HR", "SLO", "EST", "LV", "LT", "BY", "UA", "RUS"},
		"scandinavia":  {"SE", "NO", "DK", "FI", "IS"},
	}
)

// getAllCountries returns all available country codes
func getAllCountries() []string {
	return allCountries
}

// expandCountries expands regional presets to individual country codes
func expandCountries(input []string) ([]string, error) {
	var result []string
	for _, item := range input {
		lowerItem := strings.ToLower(item)
		if countries, exists := regionMap[lowerItem]; exists {
			result = append(result, countries...)
		} else {
			// Check if it's a valid country code
			found := false
			for _, validCode := range allCountries {
				if strings.ToUpper(item) == validCode {
					result = append(result, validCode)
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("invalid country/region: %s", item)
			}
		}
	}
	return removeDuplicates(result), nil
}

// removeDuplicates removes duplicate country codes
func removeDuplicates(countries []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, country := range countries {
		if !keys[country] {
			keys[country] = true
			result = append(result, country)
		}
	}
	return result
}

// loadConfigFile loads configuration from YAML file
func loadConfigFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}

// saveConfigFile saves configuration to YAML file
func saveConfigFile(config *Config, filename string) error {
	// Create a directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	return os.WriteFile(filename, data, 0600)
}

// getDefaultConfigPath returns the default configuration file path
func getDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./scdb-config.yml"
	}

	// Try XDG config directory first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "scdb", "config.yml")
	}

	// Fall back to ~/.config/scdb/config.yml
	return filepath.Join(homeDir, ".config", "scdb", "config.yml")
}

// printUsage prints enhanced usage information
func printUsage() {
	fmt.Printf("SCDB Speed Camera Downloader v1.2\n")
	fmt.Printf("Download speed camera databases from scdb.info\n\n")
	fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
	fmt.Printf("Authentication (required):\n")
	fmt.Printf("  -user string        SCDB username (or use SCDB_USER env var)\n")
	fmt.Printf("  -pass string        SCDB password (or use SCDB_PASS env var)\n\n")
	fmt.Printf("Download Options:\n")
	fmt.Printf("  -output string      Output directory (default: current dir)\n")
	fmt.Printf("  -countries string   Country codes or regions (default: all)\n")
	fmt.Printf("                        'all', country codes (NL,B,D), or regions:\n")
	fmt.Printf("                        africa, asia, europe, northamerica, southamerica, oceania\n")
	fmt.Printf("                        dach, benelux, westeurope, easteurope, scandinavia\n")
	fmt.Printf("  -fixed              Download fixed cameras (default: true)\n")
	fmt.Printf("  -mobile             Download mobile cameras (default: true)\n\n")
	fmt.Printf("Camera Configuration:\n")
	fmt.Printf("  -display int        Display type: 1-4 (default: 1)\n")
	fmt.Printf("                        1=Split all, 2=Split speed/red, 3=All in one, 4=Alt icon\n")
	fmt.Printf("  -iconsize int       Icon size: 1-5 (default: 5)\n")
	fmt.Printf("                        1=22x22, 2=24x24, 3=32x32, 4=48x48, 5=80x80 pixels\n")
	fmt.Printf("  -dangerzones        Include danger zones (default: true)\n")
	fmt.Printf("  -francedanger       France: true=danger zone, false=correct position (default: false)\n")
	fmt.Printf("  -warningtime int    Warning time in seconds, 0=disabled (default: 0)\n\n")
	fmt.Printf("Configuration File:\n")
	fmt.Printf("  -config string      Load settings from YAML file\n")
	fmt.Printf("  -saveconfig string  Save current settings to YAML file\n")
	fmt.Printf("                        Default: %s\n", getDefaultConfigPath())
	fmt.Printf("\n")
	fmt.Printf("Other Options:\n")
	fmt.Printf("  -verbose            Enable verbose output\n")
	fmt.Printf("  -help               Show this help message\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  # Download all countries with defaults\n")
	fmt.Printf("  %s -user myuser -pass mypass\n\n", os.Args[0])
	fmt.Printf("  # Download specific regions\n")
	fmt.Printf("  %s -countries \"dach,benelux\" -francedanger -warningtime 300\n\n", os.Args[0])
	fmt.Printf("  # Use config file\n")
	fmt.Printf("  %s -config ~/.config/scdb/config.yml\n\n", os.Args[0])
	fmt.Printf("Environment Variables:\n")
	fmt.Printf("  SCDB_USER     Username (alternative to -user flag)\n")
	fmt.Printf("  SCDB_PASS     Password (alternative to -pass flag)\n\n")
}

// validateConfig validates the configuration and returns any errors
func validateConfig(config *Config) error {
	// Validate required fields
	if config.Username == "" || config.Password == "" {
		return fmt.Errorf("username and password are required\nProvide via -user/-pass flags or SCDB_USER/SCDB_PASS environment variables")
	}

	// Validate flag ranges
	if config.DisplayType < 1 || config.DisplayType > 4 {
		return fmt.Errorf("display type must be 1-4 (got %d)", config.DisplayType)
	}

	if config.IconSize < 1 || config.IconSize > 5 {
		return fmt.Errorf("icon size must be 1-5 (got %d)", config.IconSize)
	}

	if config.WarningTime < 0 {
		return fmt.Errorf("warning time cannot be negative (got %d)", config.WarningTime)
	}

	// Validate that at least one download option is selected
	if !config.DownloadFixed && !config.DownloadMobile {
		return fmt.Errorf("at least one of -fixed or -mobile must be enabled")
	}

	// Validate countries
	if len(config.Countries) == 0 {
		return fmt.Errorf("no countries specified")
	}

	return nil
}

func main() {
	var config Config
	var configFile, saveConfigPath string
	var countries string

	// Custom flag handling for help
	flag.Usage = printUsage

	// Configuration file flags
	flag.StringVar(&configFile, "config", "", "Load settings from YAML config file")
	flag.StringVar(&saveConfigPath, "saveconfig", "", "Save current settings to YAML config file")

	// Parse command line flags
	flag.StringVar(&config.Username, "user", "", "SCDB username (required, or use SCDB_USER env var)")
	flag.StringVar(&config.Password, "pass", "", "SCDB password (required, or use SCDB_PASS env var)")
	flag.StringVar(&config.OutputDir, "output", ".", "Output directory for downloads")

	flag.StringVar(&countries, "countries", "all", "Comma-separated country codes, regions, or 'all' for all countries")
	flag.IntVar(&config.DisplayType, "display", 1, "Display type (1=Split all, 2=Split speed/red, 3=All in one, 4=Alt icon)")
	flag.BoolVar(&config.DangerZones, "dangerzones", true, "Include danger zones")
	flag.BoolVar(&config.FranceDangerMode, "francedanger", false, "France: true=danger zone, false=correct position")
	flag.IntVar(&config.IconSize, "iconsize", 5, "Icon size (1=22x22, 2=24x24, 3=32x32, 4=48x48, 5=80x80)")
	flag.IntVar(&config.WarningTime, "warningtime", 0, "Warning time in seconds (0=disabled, default)")

	flag.BoolVar(&config.DownloadFixed, "fixed", true, "Download fixed speed cameras")
	flag.BoolVar(&config.DownloadMobile, "mobile", true, "Download mobile speed cameras")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")

	flag.Parse()

	// Load config file if specified
	if configFile != "" {
		loadedConfig, err := loadConfigFile(configFile)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error loading config file %s: %v\n", configFile, err)
			os.Exit(1)
		}
		// Merge loaded config with command line args (command line takes precedence)
		config = *loadedConfig
		config.ConfigFile = configFile

		// Re-parse flags to override config file values
		flag.Parse()
	}

	// Use environment variables if flags not provided
	if config.Username == "" {
		config.Username = os.Getenv("SCDB_USER")
	}
	if config.Password == "" {
		config.Password = os.Getenv("SCDB_PASS")
	}

	// Parse and expand countries
	if countries == "all" {
		config.Countries = getAllCountries()
	} else {
		countryList := strings.Split(countries, ",")
		// Trim whitespace from each country/region
		for i, c := range countryList {
			countryList[i] = strings.TrimSpace(c)
		}

		expanded, err := expandCountries(countryList)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error parsing countries: %v\n", err)
			_, _ = fmt.Fprintf(os.Stderr, "\nAvailable regions: africa, asia, europe, northamerica, southamerica, oceania\n")
			_, _ = fmt.Fprintf(os.Stderr, "                   dach, benelux, westeurope, easteurope, scandinavia\n")
			os.Exit(1)
		}
		config.Countries = expanded
	}

	// Save the config file if requested (do this first to allow saving without credentials)
	if saveConfigPath != "" {
		if saveConfigPath == "default" {
			saveConfigPath = getDefaultConfigPath()
		}

		// For saving config, only validate non-credential fields
		if config.DisplayType < 1 || config.DisplayType > 4 {
			_, _ = fmt.Fprintf(os.Stderr, "Error: display type must be 1-4 (got %d)\n", config.DisplayType)
			os.Exit(1)
		}
		if config.IconSize < 1 || config.IconSize > 5 {
			_, _ = fmt.Fprintf(os.Stderr, "Error: icon size must be 1-5 (got %d)\n", config.IconSize)
			os.Exit(1)
		}
		if config.WarningTime < 0 {
			_, _ = fmt.Fprintf(os.Stderr, "Error: warning time cannot be negative (got %d)\n", config.WarningTime)
			os.Exit(1)
		}

		if err := saveConfigFile(&config, saveConfigPath); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error saving config file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configuration saved to: %s\n", saveConfigPath)
		return
	}

	// Validate configuration for running downloads
	if err := validateConfig(&config); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// Create an output directory if it doesn't exist
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Show configuration in verbose mode
	if config.Verbose {
		fmt.Println("SCDB Downloader Configuration:")
		fmt.Printf("  User: %s\n", config.Username)
		fmt.Printf("  Output: %s\n", config.OutputDir)
		fmt.Printf("  Countries: %v (%d total)\n", config.Countries, len(config.Countries))
		fmt.Printf("  Display Type: %d\n", config.DisplayType)
		fmt.Printf("  Icon Size: %d\n", config.IconSize)
		fmt.Printf("  Warning Time: %d seconds\n", config.WarningTime)
		fmt.Printf("  Danger Zones: %t\n", config.DangerZones)
		fmt.Printf("  France Danger Mode: %t\n", config.FranceDangerMode)
		fmt.Printf("  Download Fixed: %t\n", config.DownloadFixed)
		fmt.Printf("  Download Mobile: %t\n", config.DownloadMobile)
		if config.ConfigFile != "" {
			fmt.Printf("  Config File: %s\n", config.ConfigFile)
		}
		fmt.Println()
	}

	// Create a downloader and run
	downloader := NewDownloader(&config)
	if err := downloader.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		os.Exit(1)
	}

	if config.Verbose {
		fmt.Println("Downloads completed successfully!")
	}
}
