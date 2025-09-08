package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2EConfigurationFlow tests the complete configuration flow
func TestE2EConfigurationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tempDir := CreateTempDir(t, "scdb_e2e_config_test")
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test complete configuration workflow
	t.Run("Configuration save and load cycle", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "test-config.yml")

		// Create a comprehensive config
		config := &Config{
			Username:         "e2euser",
			Password:         "e2epass",
			OutputDir:        tempDir,
			Countries:        []string{"NL", "B", "D", "A", "CH"},
			DisplayType:      3,
			DangerZones:      true,
			FranceDangerMode: true,
			IconSize:         4,
			WarningTime:      600,
			DownloadFixed:    true,
			DownloadMobile:   false, // Only fixed for this test
			Verbose:          true,
		}

		// Validate original config
		err := validateConfig(config)
		AssertNoError(t, err)

		// Save config
		err = saveConfigFile(config, configPath)
		AssertNoError(t, err)
		AssertFileExists(t, configPath, 100) // At least 100 bytes

		// Load config back
		loadedConfig, err := loadConfigFile(configPath)
		AssertNoError(t, err)

		// Verify loaded config (excluding ConfigFile field)
		if loadedConfig.Username != config.Username {
			t.Errorf("Username mismatch: got %q, want %q", loadedConfig.Username, config.Username)
		}
		if loadedConfig.DisplayType != config.DisplayType {
			t.Errorf("DisplayType mismatch: got %d, want %d", loadedConfig.DisplayType, config.DisplayType)
		}
		if loadedConfig.FranceDangerMode != config.FranceDangerMode {
			t.Errorf("FranceDangerMode mismatch: got %v, want %v", loadedConfig.FranceDangerMode, config.FranceDangerMode)
		}
		if len(loadedConfig.Countries) != len(config.Countries) {
			t.Errorf("Countries length mismatch: got %d, want %d", len(loadedConfig.Countries), len(config.Countries))
		}

		// Validate loaded config
		err = validateConfig(loadedConfig)
		AssertNoError(t, err)
	})
}

// TestE2ECountryExpansion tests end-to-end country expansion scenarios
func TestE2ECountryExpansion(t *testing.T) {
	scenarios := map[string]struct {
		input          []string
		expectCount    int
		mustContain    []string
		mustNotContain []string
	}{
		"DACH_Region": {
			input:       []string{"dach"},
			expectCount: 3,
			mustContain: []string{"D", "A", "CH"},
		},
		"Benelux_Region": {
			input:       []string{"benelux"},
			expectCount: 3,
			mustContain: []string{"B", "NL", "L"},
		},
		"Europe_Large": {
			input:       []string{"europe"},
			expectCount: 35, // Approximate, may change
			mustContain: []string{"D", "FR", "GB", "NL", "I", "ES"},
		},
		"Mixed_Regions_Countries": {
			input:       []string{"dach", "benelux", "USA", "GB"},
			expectCount: 8, // 3+3+1+1, assuming no overlaps
			mustContain: []string{"D", "A", "CH", "B", "NL", "L", "USA", "GB"},
		},
		"Scandinavia_Complete": {
			input:       []string{"scandinavia"},
			expectCount: 5,
			mustContain: []string{"SE", "NO", "DK", "FI", "IS"},
		},
		"All_Countries": {
			input:       []string{"all"}, // This would be handled by getAllCountries() in real code
			expectCount: 100,             // Approximate minimum
		},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			// Skip "all" test since it requires special handling
			if scenario.input[0] == "all" {
				allCountries := getAllCountries()
				if len(allCountries) < scenario.expectCount {
					t.Errorf("getAllCountries() returned %d countries, expected at least %d", len(allCountries), scenario.expectCount)
				}
				return
			}

			result, err := expandCountries(scenario.input)
			AssertNoError(t, err)

			if len(result) < scenario.expectCount {
				t.Errorf("expandCountries() returned %d countries, expected at least %d", len(result), scenario.expectCount)
			}

			// Check required countries are present
			resultMap := make(map[string]bool)
			for _, country := range result {
				resultMap[country] = true
			}

			for _, required := range scenario.mustContain {
				if !resultMap[required] {
					t.Errorf("expandCountries() missing required country %q in result %v", required, result)
				}
			}

			for _, forbidden := range scenario.mustNotContain {
				if resultMap[forbidden] {
					t.Errorf("expandCountries() contains forbidden country %q in result %v", forbidden, result)
				}
			}

			// Verify no duplicates
			if len(result) != len(resultMap) {
				t.Errorf("expandCountries() returned duplicates: %v", result)
			}
		})
	}
}

// TestE2EValidationScenarios tests comprehensive validation scenarios
func TestE2EValidationScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		configMod   func(*Config)
		wantErr     bool
		errContains string
	}{
		{
			name: "Perfect_Configuration",
			configMod: func(c *Config) {
				// Use defaults - should be valid
			},
			wantErr: false,
		},
		{
			name: "Minimum_Valid_Config",
			configMod: func(c *Config) {
				c.Username = "u"
				c.Password = "p"
				c.Countries = []string{"NL"}
				c.DisplayType = 1
				c.IconSize = 1
				c.WarningTime = 0
				c.DownloadFixed = true
				c.DownloadMobile = false
			},
			wantErr: false,
		},
		{
			name: "Maximum_Valid_Config",
			configMod: func(c *Config) {
				c.Username = "maxuser"
				c.Password = "maxpass"
				c.Countries = getAllCountries() // All countries
				c.DisplayType = 4               // Maximum
				c.IconSize = 5                  // Maximum
				c.WarningTime = 3600            // 1 hour
				c.DownloadFixed = true
				c.DownloadMobile = true
				c.DangerZones = true
				c.FranceDangerMode = true
				c.Verbose = true
			},
			wantErr: false,
		},
		{
			name: "Regional_Combinations",
			configMod: func(c *Config) {
				c.Countries = []string{"dach", "benelux", "scandinavia"}
				// This should work after expansion
			},
			wantErr: false,
		},
		{
			name: "Edge_Case_Warning_Time_Max",
			configMod: func(c *Config) {
				c.WarningTime = 86400 // 24 hours - extreme but valid
			},
			wantErr: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Start with a valid base config
			config := CreateTestConfig()

			// Apply modifications
			scenario.configMod(config)

			// If countries need expansion, do it
			if len(config.Countries) > 0 {
				// Check if any of the countries are regions
				needsExpansion := false
				for _, country := range config.Countries {
					if _, isRegion := regionMap[strings.ToLower(country)]; isRegion {
						needsExpansion = true
						break
					}
				}

				if needsExpansion {
					expanded, err := expandCountries(config.Countries)
					if err != nil && !scenario.wantErr {
						t.Errorf("expandCountries() failed: %v", err)
						return
					} else if err == nil {
						config.Countries = expanded
					}
				}
			}

			// Validate the final config
			err := validateConfig(config)

			if (err != nil) != scenario.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, scenario.wantErr)
				return
			}

			if scenario.wantErr && scenario.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), scenario.errContains) {
					t.Errorf("validateConfig() error = %v, want error containing %q", err, scenario.errContains)
				}
			}
		})
	}
}

// TestE2EDownloaderSetup tests complete downloader setup and configuration
func TestE2EDownloaderSetup(t *testing.T) {
	tempDir := CreateTempDir(t, "scdb_e2e_downloader_test")
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test various downloader configurations
	scenarios := []struct {
		name   string
		config *Config
	}{
		{
			name: "Minimal_Setup",
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
		},
		{
			name: "Complex_Regional_Setup",
			config: &Config{
				Username:         "regionuser",
				Password:         "regionpass",
				OutputDir:        tempDir,
				Countries:        []string{"NL", "B", "D", "FR", "GB"}, // Pre-expanded
				DisplayType:      3,
				DangerZones:      true,
				FranceDangerMode: true,
				IconSize:         4,
				WarningTime:      600,
				DownloadFixed:    true,
				DownloadMobile:   false,
				Verbose:          true,
			},
		},
		{
			name: "Performance_Optimized",
			config: &Config{
				Username:         "perfuser",
				Password:         "perfpass",
				OutputDir:        tempDir,
				Countries:        []string{"D"}, // Single country for speed
				DisplayType:      1,             // Simplest display
				DangerZones:      false,         // Minimize processing
				FranceDangerMode: false,
				IconSize:         1, // Smallest icons
				WarningTime:      0, // No warnings
				DownloadFixed:    true,
				DownloadMobile:   false, // Only what's needed
				Verbose:          false,
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Validate config
			err := validateConfig(scenario.config)
			AssertNoError(t, err)

			// Create downloader
			downloader := NewDownloader(scenario.config)

			// Verify downloader setup
			if downloader == nil {
				t.Fatal("NewDownloader() returned nil")
			}

			if downloader.config == nil {
				t.Fatal("Downloader config is nil")
			}

			if downloader.client == nil {
				t.Fatal("Downloader HTTP client is nil")
			}

			// Verify configuration propagation
			if downloader.config.Username != scenario.config.Username {
				t.Errorf("Username = %q, want %q", downloader.config.Username, scenario.config.Username)
			}

			if downloader.config.Verbose != scenario.config.Verbose {
				t.Errorf("Verbose = %v, want %v", downloader.config.Verbose, scenario.config.Verbose)
			}

			if len(downloader.config.Countries) != len(scenario.config.Countries) {
				t.Errorf("Countries count = %d, want %d", len(downloader.config.Countries), len(scenario.config.Countries))
			}

			// Verify HTTP client configuration
			if downloader.client.Timeout == 0 {
				t.Error("HTTP client timeout not set")
			}

			if downloader.client.Jar == nil {
				t.Error("HTTP client cookie jar not set")
			}
		})
	}
}

// TestE2EErrorHandling tests comprehensive error handling scenarios
func TestE2EErrorHandling(t *testing.T) {
	tempDir := CreateTempDir(t, "scdb_e2e_error_test")
	defer func() { _ = os.RemoveAll(tempDir) }()

	errorScenarios := []struct {
		name        string
		setup       func() *Config
		expectError string
	}{
		{
			name: "Empty_Credentials",
			setup: func() *Config {
				config := CreateTestConfig()
				config.Username = ""
				config.Password = ""
				return config
			},
			expectError: "username and password are required",
		},
		{
			name: "Invalid_Display_Type_Low",
			setup: func() *Config {
				config := CreateTestConfig()
				config.DisplayType = 0
				return config
			},
			expectError: "display type must be 1-4",
		},
		{
			name: "Invalid_Display_Type_High",
			setup: func() *Config {
				config := CreateTestConfig()
				config.DisplayType = 5
				return config
			},
			expectError: "display type must be 1-4",
		},
		{
			name: "Invalid_Icon_Size_Low",
			setup: func() *Config {
				config := CreateTestConfig()
				config.IconSize = 0
				return config
			},
			expectError: "icon size must be 1-5",
		},
		{
			name: "Invalid_Icon_Size_High",
			setup: func() *Config {
				config := CreateTestConfig()
				config.IconSize = 6
				return config
			},
			expectError: "icon size must be 1-5",
		},
		{
			name: "Negative_Warning_Time",
			setup: func() *Config {
				config := CreateTestConfig()
				config.WarningTime = -1
				return config
			},
			expectError: "warning time cannot be negative",
		},
		{
			name: "No_Download_Options",
			setup: func() *Config {
				config := CreateTestConfig()
				config.DownloadFixed = false
				config.DownloadMobile = false
				return config
			},
			expectError: "at least one of -fixed or -mobile must be enabled",
		},
		{
			name: "Empty_Countries",
			setup: func() *Config {
				config := CreateTestConfig()
				config.Countries = []string{}
				return config
			},
			expectError: "no countries specified",
		},
		{
			name: "Invalid_Country_Code",
			setup: func() *Config {
				config := CreateTestConfig()
				// This will be tested at the expandCountries level
				return config
			},
			expectError: "", // Will be tested separately
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			config := scenario.setup()

			if scenario.name == "Invalid_Country_Code" {
				// Test invalid country expansion separately
				_, err := expandCountries([]string{"INVALID_COUNTRY"})
				if err == nil {
					t.Error("expandCountries() should fail for invalid country")
				}
				if !strings.Contains(err.Error(), "invalid country/region") {
					t.Errorf("expandCountries() error = %v, want error containing 'invalid country/region'", err)
				}
				return
			}

			err := validateConfig(config)
			if err == nil {
				t.Errorf("validateConfig() should fail for %s", scenario.name)
				return
			}

			if !strings.Contains(err.Error(), scenario.expectError) {
				t.Errorf("validateConfig() error = %v, want error containing %q", err, scenario.expectError)
			}
		})
	}
}

// TestE2EConfigPathResolution tests configuration path resolution in different environments
func TestE2EConfigPathResolution(t *testing.T) {
	// Save original environment
	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")

	defer func() {
		if originalHome == "" {
			_ = os.Unsetenv("HOME")
		} else {
			_ = os.Setenv("HOME", originalHome)
		}
		if originalXDG == "" {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG)
		}
	}()

	scenarios := []struct {
		name             string
		home             string
		xdgConfig        string
		expectedContains string
	}{
		{
			name:             "Standard_Home_Directory",
			home:             "/home/testuser",
			xdgConfig:        "",
			expectedContains: "/home/testuser/.config/scdb/config.yml",
		},
		{
			name:             "Custom_XDG_Config",
			home:             "/home/testuser",
			xdgConfig:        "/custom/config",
			expectedContains: "/custom/config/scdb/config.yml",
		},
		{
			name:             "No_Home_Directory",
			home:             "",
			xdgConfig:        "",
			expectedContains: "./scdb-config.yml",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Set up environment
			if scenario.home == "" {
				_ = os.Unsetenv("HOME")
			} else {
				_ = os.Setenv("HOME", scenario.home)
			}

			if scenario.xdgConfig == "" {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			} else {
				_ = os.Setenv("XDG_CONFIG_HOME", scenario.xdgConfig)
			}

			// Get default path
			path := getDefaultConfigPath()

			// Verify path contains expected elements
			if !strings.Contains(path, scenario.expectedContains) {
				t.Errorf("getDefaultConfigPath() = %q, want path containing %q", path, scenario.expectedContains)
			}

			// For non-fallback scenarios, ensure path is absolute
			if scenario.home != "" && !filepath.IsAbs(path) {
				t.Errorf("getDefaultConfigPath() = %q, want absolute path", path)
			}
		})
	}
}
