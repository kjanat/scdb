package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config",
			config: &Config{
				Username:         "testuser",
				Password:         "testpass",
				OutputDir:        "/tmp",
				Countries:        []string{"NL", "B"},
				DisplayType:      2,
				IconSize:         3,
				WarningTime:      300,
				DownloadFixed:    true,
				DownloadMobile:   true,
				DangerZones:      true,
				FranceDangerMode: false,
				Verbose:          false,
			},
			wantErr: false,
		},
		{
			name: "Missing username",
			config: &Config{
				Password:       "testpass",
				Countries:      []string{"NL"},
				DisplayType:    1,
				IconSize:       5,
				DownloadFixed:  true,
				DownloadMobile: true,
			},
			wantErr: true,
			errMsg:  "username and password are required",
		},
		{
			name: "Missing password",
			config: &Config{
				Username:       "testuser",
				Countries:      []string{"NL"},
				DisplayType:    1,
				IconSize:       5,
				DownloadFixed:  true,
				DownloadMobile: true,
			},
			wantErr: true,
			errMsg:  "username and password are required",
		},
		{
			name: "Invalid display type - too low",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				Countries:      []string{"NL"},
				DisplayType:    0,
				IconSize:       5,
				DownloadFixed:  true,
				DownloadMobile: true,
			},
			wantErr: true,
			errMsg:  "display type must be 1-4",
		},
		{
			name: "Invalid display type - too high",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				Countries:      []string{"NL"},
				DisplayType:    5,
				IconSize:       5,
				DownloadFixed:  true,
				DownloadMobile: true,
			},
			wantErr: true,
			errMsg:  "display type must be 1-4",
		},
		{
			name: "Invalid icon size - too low",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				Countries:      []string{"NL"},
				DisplayType:    2,
				IconSize:       0,
				DownloadFixed:  true,
				DownloadMobile: true,
			},
			wantErr: true,
			errMsg:  "icon size must be 1-5",
		},
		{
			name: "Invalid icon size - too high",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				Countries:      []string{"NL"},
				DisplayType:    2,
				IconSize:       6,
				DownloadFixed:  true,
				DownloadMobile: true,
			},
			wantErr: true,
			errMsg:  "icon size must be 1-5",
		},
		{
			name: "Negative warning time",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				Countries:      []string{"NL"},
				DisplayType:    2,
				IconSize:       3,
				WarningTime:    -1,
				DownloadFixed:  true,
				DownloadMobile: true,
			},
			wantErr: true,
			errMsg:  "warning time cannot be negative",
		},
		{
			name: "Both download options disabled",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				Countries:      []string{"NL"},
				DisplayType:    2,
				IconSize:       3,
				WarningTime:    0,
				DownloadFixed:  false,
				DownloadMobile: false,
			},
			wantErr: true,
			errMsg:  "at least one of -fixed or -mobile must be enabled",
		},
		{
			name: "No countries specified",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				Countries:      []string{},
				DisplayType:    2,
				IconSize:       3,
				WarningTime:    0,
				DownloadFixed:  true,
				DownloadMobile: true,
			},
			wantErr: true,
			errMsg:  "no countries specified",
		},
		{
			name: "Valid with only fixed download",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				Countries:      []string{"NL"},
				DisplayType:    2,
				IconSize:       3,
				WarningTime:    600,
				DownloadFixed:  true,
				DownloadMobile: false,
			},
			wantErr: false,
		},
		{
			name: "Valid with only mobile download",
			config: &Config{
				Username:       "testuser",
				Password:       "testpass",
				Countries:      []string{"NL"},
				DisplayType:    4,
				IconSize:       1,
				WarningTime:    0,
				DownloadFixed:  false,
				DownloadMobile: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestGetDefaultConfigPath(t *testing.T) {
	// Save original environment
	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")

	defer func() {
		_ = os.Setenv("HOME", originalHome)
		_ = os.Setenv("XDG_CONFIG_HOME", originalXDG)
	}()

	tests := []struct {
		name           string
		home           string
		xdgConfig      string
		expectedSuffix string
	}{
		{
			name:           "With XDG_CONFIG_HOME set",
			home:           "/home/testuser",
			xdgConfig:      "/home/testuser/.config",
			expectedSuffix: "/.config/scdb/config.yml",
		},
		{
			name:           "Without XDG_CONFIG_HOME",
			home:           "/home/testuser",
			xdgConfig:      "",
			expectedSuffix: "/.config/scdb/config.yml",
		},
		{
			name:           "Custom XDG_CONFIG_HOME",
			home:           "/home/testuser",
			xdgConfig:      "/custom/config",
			expectedSuffix: "/scdb/config.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("HOME", tt.home)
			if tt.xdgConfig != "" {
				_ = os.Setenv("XDG_CONFIG_HOME", tt.xdgConfig)
			} else {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			}

			path := getDefaultConfigPath()

			if !strings.HasSuffix(path, tt.expectedSuffix) {
				t.Errorf("getDefaultConfigPath() = %q, want suffix %q", path, tt.expectedSuffix)
			}

			// Ensure path is absolute
			if !filepath.IsAbs(path) {
				t.Errorf("getDefaultConfigPath() = %q, want absolute path", path)
			}
		})
	}

	// Test fallback when HOME is not set
	t.Run("Fallback when HOME not set", func(t *testing.T) {
		_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("XDG_CONFIG_HOME")

		path := getDefaultConfigPath()
		expected := "./scdb-config.yml"

		if path != expected {
			t.Errorf("getDefaultConfigPath() = %q, want %q", path, expected)
		}
	})
}

func TestLoadConfigFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "scdb_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name        string
		fileContent string
		expected    *Config
		wantErr     bool
		errMsg      string
	}{
		{
			name: "Valid config file",
			fileContent: `username: "testuser"
password: "testpass"
output_dir: "./downloads"
countries:
- NL
- B
- D
display_type: 3
danger_zones: true
france_danger_mode: false
icon_size: 4
warning_time: 300
download_fixed: true
download_mobile: true
verbose: false`,
			expected: &Config{
				Username:         "testuser",
				Password:         "testpass",
				OutputDir:        "./downloads",
				Countries:        []string{"NL", "B", "D"},
				DisplayType:      3,
				DangerZones:      true,
				FranceDangerMode: false,
				IconSize:         4,
				WarningTime:      300,
				DownloadFixed:    true,
				DownloadMobile:   true,
				Verbose:          false,
			},
			wantErr: false,
		},
		{
			name: "Empty credentials (should be filled by env vars)",
			fileContent: `username: ""
password: ""
output_dir: "."
countries:
- "SE"
- "NO"
display_type: 1
danger_zones: true
france_danger_mode: true
icon_size: 5
warning_time: 600
download_fixed: true
download_mobile: true
verbose: true`,
			expected: &Config{
				Username:         "",
				Password:         "",
				OutputDir:        ".",
				Countries:        []string{"SE", "NO"},
				DisplayType:      1,
				DangerZones:      true,
				FranceDangerMode: true,
				IconSize:         5,
				WarningTime:      600,
				DownloadFixed:    true,
				DownloadMobile:   true,
				Verbose:          true,
			},
			wantErr: false,
		},
		{
			name:        "Invalid YAML syntax",
			fileContent: "invalid: yaml: content: [",
			expected:    nil,
			wantErr:     true,
			errMsg:      "error parsing config file",
		},
		{
			name:        "Empty file",
			fileContent: "",
			expected:    &Config{}, // Default zero values
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, "config.yml")
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0600)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			got, err := loadConfigFile(testFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfigFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("loadConfigFile() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("loadConfigFile() = %+v, want %+v", got, tt.expected)
			}
		})
	}

	// Test file not found
	t.Run("File not found", func(t *testing.T) {
		_, err := loadConfigFile("/nonexistent/file.yml")
		if err == nil {
			t.Error("loadConfigFile() expected error for nonexistent file, got nil")
		}
	})
}

func TestSaveConfigFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "scdb_config_save_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{
		Username:         "testuser",
		Password:         "testpass",
		OutputDir:        "./downloads",
		Countries:        []string{"NL", "B", "D"},
		DisplayType:      3,
		DangerZones:      true,
		FranceDangerMode: false,
		IconSize:         4,
		WarningTime:      300,
		DownloadFixed:    true,
		DownloadMobile:   true,
		Verbose:          false,
	}

	t.Run("Save to new file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "save_test.yml")

		err := saveConfigFile(config, testFile)
		if err != nil {
			t.Errorf("saveConfigFile() error = %v", err)
			return
		}

		// Verify file was created
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Errorf("saveConfigFile() did not create file %s", testFile)
			return
		}

		// Verify file contents by loading it back
		loaded, err := loadConfigFile(testFile)
		if err != nil {
			t.Errorf("Failed to load saved config: %v", err)
			return
		}

		// ConfigFile field should not be serialized, so exclude from comparison
		loaded.ConfigFile = config.ConfigFile

		if !reflect.DeepEqual(loaded, config) {
			t.Errorf("Loaded config = %+v, want %+v", loaded, config)
		}
	})

	t.Run("Save with directory creation", func(t *testing.T) {
		nestedDir := filepath.Join(tempDir, "nested", "dir")
		testFile := filepath.Join(nestedDir, "config.yml")

		err := saveConfigFile(config, testFile)
		if err != nil {
			t.Errorf("saveConfigFile() error = %v", err)
			return
		}

		// Verify directory and file were created
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Errorf("saveConfigFile() did not create file %s", testFile)
		}
	})

	t.Run("Invalid directory permissions", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		// Try to save to a location where we can't create directories
		testFile := "/root/cannot_create/config.yml"

		err := saveConfigFile(config, testFile)
		if err == nil {
			t.Error("saveConfigFile() expected error for invalid directory, got nil")
		}
	})
}

func TestConfigRoundTrip(t *testing.T) {
	// Test that save -> load produces identical config
	tempDir, err := os.MkdirTemp("", "scdb_config_roundtrip_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	original := &Config{
		Username:         "roundtrip_user",
		Password:         "roundtrip_pass",
		OutputDir:        "/tmp/scdb",
		Countries:        []string{"NL", "B", "D", "FR", "GB"},
		DisplayType:      2,
		DangerZones:      true,
		FranceDangerMode: true,
		IconSize:         3,
		WarningTime:      450,
		DownloadFixed:    true,
		DownloadMobile:   false,
		Verbose:          true,
	}

	testFile := filepath.Join(tempDir, "roundtrip.yml")

	// Save
	err = saveConfigFile(original, testFile)
	if err != nil {
		t.Fatalf("saveConfigFile() error = %v", err)
	}

	// Load
	loaded, err := loadConfigFile(testFile)
	if err != nil {
		t.Fatalf("loadConfigFile() error = %v", err)
	}

	// Compare (ConfigFile field is not serialized)
	loaded.ConfigFile = original.ConfigFile

	if !reflect.DeepEqual(loaded, original) {
		t.Errorf("Round trip failed:\nOriginal: %+v\nLoaded:   %+v", original, loaded)
	}
}
