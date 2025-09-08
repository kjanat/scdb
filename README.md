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

# Regional presets and custom settings
./scdb-downloader \
  -output ./downloads \
  -countries "dach,benelux" \
  -display 3 \
  -iconsize 4 \
  -francedanger \
  -warningtime 300 \
  -fixed \
  -mobile \
  -verbose

# Using configuration files
./scdb-downloader -config ~/.config/scdb/config.yml

# Save current settings as default config
./scdb-downloader -countries "westeurope" -francedanger -saveconfig default
```

## Command Line Options

| Flag            | Description                                                   | Default           |
|-----------------|---------------------------------------------------------------|-------------------|
| `-user`         | SCDB username (required, or use SCDB_USER env var)            | -                 |
| `-pass`         | SCDB password (required, or use SCDB_PASS env var)            | -                 |
| `-output`       | Output directory for downloads                                | `.` (current dir) |
| `-countries`    | Comma-separated country codes or 'all'                        | `all`             |
| `-display`      | Display type (see below)                                      | `1`               |
| `-dangerzones`  | Include danger zones                                          | `true`            |
| `-iconsize`     | Icon size (see below)                                         | `5`               |
| `-warningtime`  | Warning time in seconds (0=disabled)                          | `0`               |
| `-francedanger` | France danger zones: true=danger zone, false=correct position | `false`           |
| `-config`       | Load settings from YAML configuration file                    | -                 |
| `-saveconfig`   | Save current settings to YAML configuration file              | -                 |
| `-fixed`        | Download fixed speed cameras                                  | `true`            |
| `-mobile`       | Download mobile speed cameras                                 | `true`            |
| `-verbose`      | Enable verbose output                                         | `false`           |

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

- `0` = Disabled (default)—No warning time/distance
- Any positive integer = Warning time in seconds before reaching the camera
- This value is passed to the SCDB system and may affect the database content

## Country Codes and Regional Presets

The application supports all 110+ countries/territories available on SCDB.

### Individual Country Codes

- `NL` = Netherlands, `B` = Belgium, `D` = Germany, `FR` = France
- `GB` = United Kingdom, `USA` = United States, `A` = Austria, `CH` = Switzerland
- `I` = Italy, `ES` = Spain, `P` = Portugal, `PL` = Poland
- And 100+ more countries and territories...

### Regional Presets

**Continental Regions:**

- `africa` = All African countries with speed cameras
- `asia` = All Asian countries with speed cameras
- `europe` = All European countries with speed cameras
- `northamerica` = North American countries (USA, Canada, Mexico, etc.)
- `southamerica` = South American countries (Brazil, Argentina, Chile, etc.)
- `oceania` = Australia, New Zealand, Fiji

**European Sub-regions:**

- `dach` = Germany, Austria, Switzerland
- `benelux` = Belgium, Netherlands, Luxembourg
- `westeurope` = Western European countries
- `easteurope` = Eastern European countries
- `scandinavia` = Sweden, Norway, Denmark, Finland, Iceland

**Examples:**

```bash
# Single region
./scdb-downloader -countries "dach"

# Multiple regions
./scdb-downloader -countries "dach,benelux,scandinavia"

# Mix regions and individual countries
./scdb-downloader -countries "dach,FR,GB,USA"
```

### France-Specific Options

- `-francedanger false` = Display correct camera position (default)
- `-francedanger true` = Display position as danger zone (regulatory compliance)

### Warning Time

- `0` = Disabled (default)
- `>0` = Warning time in seconds before reaching camera location

## Configuration Files

### YAML Configuration

Save and load settings using YAML configuration files:

```yaml
# Example ~/.config/scdb/config.yml
username: "" # Leave empty, use environment variables
password: "" # Leave empty, use environment variables
output_dir: "./downloads"
countries:
  - D
  - A
  - CH
display_type: 3
danger_zones: true
france_danger_mode: true
icon_size: 4
warning_time: 300
download_fixed: true
download_mobile: true
verbose: false
```

### Config File Commands

```bash
# Save current settings to default location
./scdb-downloader -countries "dach" -francedanger -saveconfig default

# Save to custom location
./scdb-downloader -countries "benelux" -saveconfig ~/my-config.yml

# Load from config file
./scdb-downloader -config ~/.config/scdb/config.yml

# Override config file settings
./scdb-downloader -config ~/my-config.yml -countries "all" -verbose
```

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
- Active SCDB.info account with a valid subscription
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
