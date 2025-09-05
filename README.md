# SCDB Speed Camera Downloader (Go Implementation)

A Go implementation for downloading speed camera databases from scdb.info (flitspalen.nl).

## Features

- Downloads both fixed and mobile speed camera databases
- Configurable country selection
- Multiple display type options
- Customizable icon sizes
- Session management with cookie handling
- Secure HTTPS connections

## Installation

```bash
go build -o scdb-downloader scdb_downloader.go
```

## Usage

### Basic Usage

Download all databases with default settings:
```bash
# Using command line flags
./scdb-downloader -user your_username -pass "your_password"

# Using environment variables (recommended for security)
export SCDB_USER=your_username
export SCDB_PASS=your_password
./scdb-downloader
```

### Advanced Options

```bash
# Using environment variables for credentials
export SCDB_USER=your_username  
export SCDB_PASS=your_password

./scdb-downloader \
  -output ./downloads \
  -countries "NL,B,D,FR" \
  -display 1 \
  -iconsize 5 \
  -warningtime 300 \
  -fixed \
  -mobile \
  -verbose
```

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-user` | SCDB username (required, or use SCDB_USER env var) | - |
| `-pass` | SCDB password (required, or use SCDB_PASS env var) | - |
| `-output` | Output directory for downloads | `.` (current dir) |
| `-countries` | Comma-separated country codes or 'all' | `all` |
| `-display` | Display type (see below) | `1` |
| `-dangerzones` | Include danger zones | `true` |
| `-iconsize` | Icon size (see below) | `5` |
| `-warningtime` | Warning time in seconds (0=disabled) | `0` |
| `-fixed` | Download fixed speed cameras | `true` |
| `-mobile` | Download mobile speed cameras | `true` |
| `-verbose` | Enable verbose output | `false` |

### Display Types
- `1` = Split into all categories (multiple files)
- `2` = Split into speed cameras & redlights (2 files)
- `3` = All safety cameras in one category (1 file)
- `4` = All safety cameras in one category (alternative icon)

### Icon Sizes
- `1` = 22x22 pixels (4 bit BMP)
- `2` = 24x24 pixels (8 bit BMP)
- `3` = 32x32 pixels (8 bit BMP)
- `4` = 48x48 pixels (8 bit BMP)
- `5` = 80x80 pixels (8 bit BMP)

### Warning Time
The `-warningtime` option allows you to set a warning distance/time for speed cameras:
- `0` = Disabled (default) - No warning time/distance
- Any positive integer = Warning time in seconds before reaching the camera
- This value is passed to the SCDB system and may affect the database content

## Country Codes

The application supports all countries available on SCDB. Use `all` to download data for all countries, or specify individual country codes:

- `NL` = Netherlands
- `B` = Belgium
- `D` = Germany
- `FR` = France
- `GB` = United Kingdom
- `USA` = United States
- And many more...

## Output Files

The downloader creates two files in the output directory:
- `garmin.zip` - Fixed speed camera database
- `garmin-mobile.zip` - Mobile speed camera database

## Security Notes

- The application uses HTTPS for all connections
- Credentials are sent over encrypted connections
- Session cookies are managed automatically
- No credentials are stored locally

## Requirements

- Go 1.16 or higher
- Active SCDB.info account with valid subscription
- Internet connection

## Building from Source

```bash
# Clone or create the project
go mod init scdb-downloader
go mod tidy
go build -o scdb-downloader scdb_downloader.go
```

## Example Script

Create a script for automated downloads:

```bash
#!/bin/bash
# download-cameras.sh

OUTPUT_DIR="$HOME/Downloads/scdb-$(date +%Y%m%d)"
mkdir -p "$OUTPUT_DIR"

./scdb-downloader \
  -user "$SCDB_USER" \
  -pass "$SCDB_PASS" \
  -output "$OUTPUT_DIR" \
  -countries all \
  -verbose

echo "Downloads saved to: $OUTPUT_DIR"
```

## Troubleshooting

1. **Login fails**: Verify your credentials are correct
2. **Download fails**: Check your subscription is active
3. **Network errors**: The tool handles SSL certificates automatically
4. **Empty files**: Enable verbose mode to see server responses

## License

This is a third-party implementation for personal use. Respect SCDB.info's terms of service.