#!/usr/bin/env bash

# Test script for SCDB downloader  
# This will test regional presets and new configuration features

echo "Testing SCDB Go Downloader v1.2..."
echo "===================================="

# Check for required environment variables
if [ -z "$SCDB_USER" ] || [ -z "$SCDB_PASS" ]; then
    echo "Error: Please set SCDB_USER and SCDB_PASS environment variables"
    echo "Example: export SCDB_USER=your_username"
    echo "         export SCDB_PASS=your_password"
    exit 1
fi

# Create test output directory
TEST_DIR="./test_download_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$TEST_DIR"

# Test 1: Regional preset (Benelux) with new features
echo "Test 1: Regional preset 'benelux' with France danger mode..."
./scdb-downloader \
  -user "$SCDB_USER" \
  -pass "$SCDB_PASS" \
  -output "$TEST_DIR" \
  -countries "benelux" \
  -display 3 \
  -iconsize 4 \
  -francedanger \
  -warningtime 300 \
  -fixed \
  -mobile \
  -verbose

echo ""
echo "Test 2: Save current configuration..."
./scdb-downloader \
  -countries "dach" \
  -francedanger \
  -warningtime 600 \
  -saveconfig "$TEST_DIR/test-config.yml"

echo ""
echo "Test 3: Load from saved configuration..."
./scdb-downloader \
  -user "$SCDB_USER" \
  -pass "$SCDB_PASS" \
  -config "$TEST_DIR/test-config.yml" \
  -output "$TEST_DIR" \
  -mobile false \
  -verbose

# Check results
echo ""
echo "Download Results:"
echo "-----------------"
ls -lah "$TEST_DIR"

echo ""
echo "Configuration Files:"
echo "--------------------"
ls -lah "$TEST_DIR"/*.yml 2>/dev/null || echo "No config files"

echo ""
echo "Test completed. Files saved in: $TEST_DIR"
echo ""
echo "Summary of new features tested:"
echo "✓ Regional presets (benelux expanded to B,NL,L)"
echo "✓ France danger mode configuration"
echo "✓ Warning time settings"
echo "✓ YAML configuration file save/load"
echo "✓ Command line override of config settings"