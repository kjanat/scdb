package main

import (
	"reflect"
	"testing"
)

func TestGetAllCountries(t *testing.T) {
	countries := getAllCountries()

	// Test that we get a reasonable number of countries
	if len(countries) < 100 {
		t.Errorf("Expected at least 100 countries, got %d", len(countries))
	}

	// Test that some known countries are present
	expectedCountries := []string{"D", "NL", "B", "FR", "USA", "GB"}
	for _, expected := range expectedCountries {
		found := false
		for _, country := range countries {
			if country == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected country %s not found in countries list", expected)
		}
	}

	// Test that there are no duplicates
	seen := make(map[string]bool)
	for _, country := range countries {
		if seen[country] {
			t.Errorf("Duplicate country code found: %s", country)
		}
		seen[country] = true
	}
}

func TestExpandCountries(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
		wantErr  bool
	}{
		{
			name:     "Single valid country",
			input:    []string{"NL"},
			expected: []string{"NL"},
			wantErr:  false,
		},
		{
			name:     "Multiple valid countries",
			input:    []string{"NL", "B", "D"},
			expected: []string{"NL", "B", "D"},
			wantErr:  false,
		},
		{
			name:     "DACH region preset",
			input:    []string{"dach"},
			expected: []string{"D", "A", "CH"},
			wantErr:  false,
		},
		{
			name:     "Benelux region preset",
			input:    []string{"benelux"},
			expected: []string{"B", "NL", "L"},
			wantErr:  false,
		},
		{
			name:     "Scandinavia region preset",
			input:    []string{"scandinavia"},
			expected: []string{"SE", "NO", "DK", "FI", "IS"},
			wantErr:  false,
		},
		{
			name:     "Mixed countries and regions",
			input:    []string{"dach", "FR", "GB"},
			expected: []string{"D", "A", "CH", "FR", "GB"},
			wantErr:  false,
		},
		{
			name:     "Case insensitive region",
			input:    []string{"DACH", "Benelux"},
			expected: []string{"D", "A", "CH", "B", "NL", "L"},
			wantErr:  false,
		},
		{
			name:     "Case insensitive country",
			input:    []string{"nl", "gb", "usa"},
			expected: []string{"NL", "GB", "USA"},
			wantErr:  false,
		},
		{
			name:     "Duplicate countries should be deduplicated",
			input:    []string{"NL", "NL", "B"},
			expected: []string{"NL", "B"},
			wantErr:  false,
		},
		{
			name:     "Region with overlapping countries",
			input:    []string{"dach", "D"},
			expected: []string{"D", "A", "CH"},
			wantErr:  false,
		},
		{
			name:     "Invalid country code",
			input:    []string{"INVALID"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Invalid region name",
			input:    []string{"unknownregion"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Empty input",
			input:    []string{},
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "Whitespace in country names",
			input:    []string{" NL ", " B "},
			expected: nil,
			wantErr:  true, // Current implementation doesn't trim whitespace
		},
		{
			name:     "Europe region (large set)",
			input:    []string{"europe"},
			expected: []string{"AND", "A", "BY", "B", "BIH", "BG", "HR", "CY", "CZ", "DK", "EST", "FI", "FR", "GE", "D", "GBZ", "GR", "H", "IS", "IRL", "I", "LV", "RL", "LI", "LT", "L", "M", "MK", "NO", "PL", "P", "RO", "RUS", "RSM", "SRB", "SK", "SLO", "ES", "SE", "CH", "TR", "UA", "GB"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandCountries(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("expandCountries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Handle empty slice comparison
			if len(got) == 0 && len(tt.expected) == 0 {
				return // Both empty, test passes
			}

			// Sort both slices for comparison since order might vary
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("expandCountries() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "No duplicates",
			input:    []string{"A", "B", "C"},
			expected: []string{"A", "B", "C"},
		},
		{
			name:     "With duplicates",
			input:    []string{"A", "B", "A", "C", "B"},
			expected: []string{"A", "B", "C"},
		},
		{
			name:     "All same elements",
			input:    []string{"A", "A", "A"},
			expected: []string{"A"},
		},
		{
			name:     "Empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single element",
			input:    []string{"A"},
			expected: []string{"A"},
		},
		{
			name:     "Many duplicates",
			input:    []string{"NL", "B", "NL", "D", "B", "NL", "FR"},
			expected: []string{"NL", "B", "D", "FR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeDuplicates(tt.input)

			// Check length first
			if len(got) != len(tt.expected) {
				t.Errorf("removeDuplicates() length = %d, want %d", len(got), len(tt.expected))
				return
			}

			// Convert to sets for comparison (order doesn't matter)
			gotSet := make(map[string]bool)
			expectedSet := make(map[string]bool)

			for _, item := range got {
				gotSet[item] = true
			}
			for _, item := range tt.expected {
				expectedSet[item] = true
			}

			if !reflect.DeepEqual(gotSet, expectedSet) {
				t.Errorf("removeDuplicates() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Test edge cases and error conditions
func TestExpandCountriesEdgeCases(t *testing.T) {
	// Test all available regions to ensure they expand correctly
	regions := []string{"africa", "asia", "europe", "northamerica", "southamerica", "oceania", "dach", "benelux", "westeurope", "easteurope", "scandinavia"}

	for _, region := range regions {
		t.Run("region_"+region, func(t *testing.T) {
			result, err := expandCountries([]string{region})
			if err != nil {
				t.Errorf("Region %s should be valid, got error: %v", region, err)
			}
			if len(result) == 0 {
				t.Errorf("Region %s should expand to at least one country", region)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkExpandCountries(b *testing.B) {
	input := []string{"dach", "benelux", "scandinavia", "FR", "GB", "USA"}

	for i := 0; i < b.N; i++ {
		_, err := expandCountries(input)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkRemoveDuplicates(b *testing.B) {
	// Create input with many duplicates
	input := make([]string, 100)
	countries := []string{"NL", "B", "D", "FR", "GB"}
	for i := range input {
		input[i] = countries[i%len(countries)]
	}

	for i := 0; i < b.N; i++ {
		removeDuplicates(input)
	}
}
