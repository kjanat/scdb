# SCDB Downloader Test Documentation

Comprehensive testing framework for the SCDB Speed Camera Downloader with systematic test coverage across all components.

## Test Architecture

### Testing Strategy Overview

Our testing approach follows a **multi-layered pyramid** strategy:

1. **Unit Tests (70%)** - Fast, isolated, pure function testing
2. **Integration Tests (20%)** - HTTP workflows with mocking  
3. **E2E Tests (10%)** - Complete scenarios and workflows

### Test Files Structure

```
├── countries_test.go          # Country/region mapping logic
├── config_test.go            # Configuration management
├── scdb_downloader_test.go   # HTTP client and download workflows
├── e2e_test.go              # End-to-end scenarios
├── testhelpers_test.go      # Shared test utilities and mocks
└── TEST_DOCUMENTATION.md    # This file
```

## Test Categories

### 1. Unit Tests (`*_test.go`)

**Pure Function Testing - No Dependencies**

**`countries_test.go`** - Country and region mapping logic:
- `TestGetAllCountries()` - Validates country list completeness and integrity
- `TestExpandCountries()` - Regional preset expansion with 15+ scenarios
- `TestRemoveDuplicates()` - Deduplication logic with edge cases
- **Coverage**: >95% of country mapping functions

**Key Test Scenarios**:
```go
// Regional preset expansion
"dach" → ["D", "A", "CH"]
"benelux" → ["B", "NL", "L"] 
"scandinavia" → ["SE", "NO", "DK", "FI", "IS"]

// Mixed scenarios
["dach", "FR", "GB"] → ["D", "A", "CH", "FR", "GB"]

// Error cases
["INVALID"] → error "invalid country/region"
```

**`config_test.go`** - Configuration management:
- `TestValidateConfig()` - Comprehensive validation with 12+ scenarios
- `TestLoadConfigFile()` / `TestSaveConfigFile()` - YAML serialization
- `TestGetDefaultConfigPath()` - XDG-compliant path resolution
- **Coverage**: >90% of configuration functions

**Key Test Scenarios**:
```go
// Validation scenarios
✅ Valid config with all options
❌ Missing credentials → "username and password are required"
❌ Invalid display type → "display type must be 1-4"
❌ Both downloads disabled → "at least one must be enabled"

// Path resolution
$HOME="/home/user" → "/home/user/.config/scdb/config.yml"
$XDG_CONFIG_HOME="/custom" → "/custom/scdb/config.yml" 
No $HOME → "./scdb-config.yml"
```

### 2. Integration Tests

**HTTP Workflows with Mocking**

**`scdb_downloader_test.go`** - Core downloader functionality:
- `TestNewDownloader()` - HTTP client configuration (TLS, cookies, timeouts)
- `TestSCDBDownloader_saveResponseToFile()` - File saving with content validation
- `TestSCDBDownloader_Run()` - Workflow orchestration
- **Coverage**: >80% of HTTP-related code

**Mock Infrastructure** (`testhelpers_test.go`):
- `MockSCDBServer` - Complete SCDB API simulation
  - Login page with CSRF token generation
  - Form-based authentication with validation
  - Fixed/mobile download endpoints
  - Configurable failure scenarios

**Key Test Scenarios**:
```go
// HTTP client setup
✅ Cookie jar configuration
✅ TLS InsecureSkipVerify for self-signed certs
✅ 5-minute timeout setting

// Response handling  
✅ "application/zip" → Save file
✅ "application/octetstream" → Save file (matches real server)
❌ "text/html" → Error "unexpected response"

// CSRF token extraction
✅ Extract 40-character hex tokens from HTML
❌ No token found → Login failure
```

### 3. End-to-End Tests

**Complete Workflow Testing**

**`e2e_test.go`** - Comprehensive scenario testing:
- `TestE2EConfigurationFlow()` - Complete config save/load cycle
- `TestE2ECountryExpansion()` - Large-scale regional expansion scenarios  
- `TestE2EValidationScenarios()` - Edge cases and comprehensive validation
- `TestE2EDownloaderSetup()` - Complete downloader initialization
- `TestE2EErrorHandling()` - Systematic error scenario coverage

**Key Test Scenarios**:
```go
// Configuration flow
Create config → Save to YAML → Load back → Validate identical

// Large-scale expansion
"europe" → 40+ countries including D, FR, GB, NL, I, ES
"dach,benelux,scandinavia" → 11 countries with no duplicates

// Error handling matrix
10+ error scenarios covering all validation paths
Credential errors, range errors, logical errors
```

## Test Infrastructure

### Mock Server (`MockSCDBServer`)

**Complete SCDB API Simulation**:
```go
// Endpoints
GET  /en/login/           → HTML with CSRF token
POST /en/login/           → Authentication with validation  
POST /my/downloadsection  → Fixed camera download
POST /intern/download/garmin-mobile.zip → Mobile download

// Features
- CSRF token generation and validation
- Form data validation (countries, display options)
- Configurable failures for testing error scenarios
- Request counting and statistics
```

### Test Helpers (`testhelpers_test.go`)

**Shared Test Utilities**:
```go
CreateTestConfig()          // Standard valid config for testing
CreateTestDownloader()      // Pre-configured downloader instance
CreateTempDir()            // Temporary directory management
AssertFileExists()         // File existence and size validation
AssertErrorContains()      // Error message validation
MockHTMLResponse()         // HTML response generation
ValidateFormData()         // HTTP form validation
CountriesTestData()        // Standard country test datasets
```

### Quality Assurance Utilities
```go
// Assertion helpers
AssertNoError(t, err)                    // Clean error checking
AssertFileExists(t, path, minSize)       // File validation
AssertFileNotExists(t, path)             // Non-existence verification
AssertErrorContains(t, err, substring)   // Error message validation

// Test data builders
CreateTestConfig()           // Valid baseline configuration  
CountriesTestData()         // Standard regional test data
MockHTMLResponse(token)     // CSRF-enabled HTML responses
```

## Running Tests

### Command Line Usage

```bash
# All tests
go test ./...

# Unit tests only (fast)
go test -short ./...

# Integration tests
go test -run "TestSCDBDownloader|TestMock" ./...

# E2E tests  
go test -run "TestE2E" ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Verbose output
go test -v ./...

# Benchmarks
go test -bench=. ./...
```

### Makefile Targets

```bash
make test              # All tests
make test-unit         # Unit tests only
make test-integration  # Integration tests with mocks
make test-e2e         # End-to-end scenarios
make test-coverage    # Tests with coverage report
make test-verbose     # Detailed test output
make test-bench       # Benchmark tests
```

## Test Coverage Goals

### Coverage Targets
- **Unit Tests**: >95% coverage on pure functions
- **Integration Tests**: >80% coverage on HTTP/file operations
- **Overall**: >85% total coverage
- **Critical Paths**: 100% coverage on authentication and download flows

### Coverage by Component
```
countries.go     →  >95%  (Pure functions)
config.go        →  >90%  (File I/O + validation) 
scdb_downloader.go → >80%  (HTTP + complex workflows)
main.go          →  >70%  (CLI parsing, difficult to test)
```

## Test Quality Standards

### Test Naming Convention
```go
TestFunction_Scenario_ExpectedOutcome
TestExpandCountries_ValidRegion_ReturnsCountryList
TestValidateConfig_MissingCredentials_ReturnsError
TestE2EConfigurationFlow_SaveAndLoad_IdenticalResults
```

### Test Organization
```go
// Table-driven tests for multiple scenarios
tests := []struct {
    name     string
    input    []string
    expected []string
    wantErr  bool
}{
    {"Valid region", []string{"dach"}, []string{"D", "A", "CH"}, false},
    {"Invalid region", []string{"invalid"}, nil, true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test implementation
    })
}
```

### Error Testing Standards
- **Test all error paths**: Every error return must have a test
- **Validate error messages**: Check that errors contain expected text
- **Test error chains**: Verify wrapped errors propagate correctly

## Continuous Integration

### GitHub Actions Pipeline (`.github/workflows/test.yml`)
1. **Multi-version testing**: Go 1.19, 1.20, 1.21
2. **Code quality gates**: gofmt, go vet, golangci-lint
3. **Test execution**: Unit → Integration → E2E
4. **Coverage reporting**: Codecov integration
5. **Security scanning**: gosec, nancy, govulncheck
6. **Cross-platform builds**: Linux, macOS, Windows

### Quality Gates
```yaml
✅ Code formatting (gofmt)
✅ Static analysis (go vet)  
✅ Linting (golangci-lint)
✅ Unit tests (>95% pure functions)
✅ Integration tests (>80% HTTP workflows)
✅ E2E tests (critical scenarios)
✅ Security scan (gosec)
✅ Dependency check (nancy + govulncheck)
✅ Cross-platform builds
```

## Test Data and Fixtures

### Sample Configurations
```yaml
# Minimal valid config
username: "testuser"
password: "testpass"
countries: ["NL"]
display_type: 1
icon_size: 5
download_fixed: true
download_mobile: true

# Comprehensive config
countries: ["NL", "B", "D", "FR", "GB"]
display_type: 3
danger_zones: true
france_danger_mode: true
icon_size: 4
warning_time: 600
verbose: true
```

### Mock HTML Responses
```html
<!-- Login page with CSRF token -->
<form method="POST" action="/en/login/">
    <input type="hidden" name="abcdef1234567890abcdef1234567890abcdef12" 
           value="abcdef1234567890abcdef1234567890abcdef12">
    <input type="text" name="u_name" placeholder="Username">
    <input type="password" name="u_password" placeholder="Password">
</form>
```

## Performance Testing

### Benchmark Tests
```go
BenchmarkExpandCountries    // Country expansion performance
BenchmarkRemoveDuplicates   // Deduplication algorithm performance
```

### Performance Expectations
- Country expansion: <1ms for typical regions
- Config load/save: <10ms for standard configs
- CSRF token extraction: <1ms for typical HTML

## Debugging and Troubleshooting

### Test Debugging
```bash
# Run specific test with verbose output
go test -v -run TestExpandCountries_DACH_Region

# Debug with coverage
go test -cover -v -run TestE2EConfigurationFlow

# Race condition detection  
go test -race ./...

# CPU profiling
go test -cpuprofile cpu.prof -bench .
```

### Common Issues
1. **Temp directory cleanup**: Always use `defer os.RemoveAll(tempDir)`
2. **Environment isolation**: Save/restore environment variables in tests
3. **Mock server lifecycle**: Ensure proper server start/stop in tests
4. **File permissions**: Use appropriate permissions for test files (0600 for configs)

## Future Test Enhancements

### Potential Improvements
1. **Property-based testing**: Use `gopter` for broader input validation
2. **Mutation testing**: Verify test quality with `go-mutesting`
3. **Performance regression tests**: Track performance metrics over time
4. **Integration with real SCDB**: Optional tests with real credentials
5. **Docker-based E2E**: Complete environment simulation

### Test Maintenance
- **Review quarterly**: Update test scenarios as features evolve
- **Monitor coverage**: Maintain >85% overall coverage
- **Performance tracking**: Monitor test execution time
- **Dependency updates**: Keep test libraries current