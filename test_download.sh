#!/bin/bash

# Test script for SCDB downloader
# This will download just Netherlands data as a test

echo "Testing SCDB Go Downloader..."
echo "=============================="

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

# Run the downloader with test parameters
./scdb-downloader \
  -user "$SCDB_USER" \
  -pass "$SCDB_PASS" \
  -output "$TEST_DIR" \
  -countries "NL" \
  -display 1 \
  -iconsize 5 \
  -warningtime 0 \
  -fixed \
  -mobile \
  -verbose

# Check results
echo ""
echo "Download Results:"
echo "-----------------"
ls -lah "$TEST_DIR"

echo ""
echo "Test completed. Files saved in: $TEST_DIR"