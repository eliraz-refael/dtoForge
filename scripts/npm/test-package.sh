#!/bin/bash

# Test script for npm package before publishing
# This script builds a test package and verifies it works correctly

set -e

echo "🧪 DtoForge NPM Package Test Script"
echo "===================================="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"

# Create temp directory for testing
TEST_DIR=$(mktemp -d)
NPM_PACKAGE_DIR="$TEST_DIR/npm-package"
TEST_PROJECT_DIR="$TEST_DIR/test-project"

echo ""
echo "📁 Test directory: $TEST_DIR"

# Cleanup on exit
cleanup() {
  echo ""
  echo "🧹 Cleaning up test directory..."
  rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# Step 1: Build the binary for current platform
echo ""
echo "1️⃣  Building binary for current platform..."
cd "$PROJECT_ROOT"
go build -o "$TEST_DIR/dtoforge-test" .
if [ $? -eq 0 ]; then
  echo -e "${GREEN}✅ Binary built successfully${NC}"
else
  echo -e "${RED}❌ Failed to build binary${NC}"
  exit 1
fi

# Step 2: Create npm package structure
echo ""
echo "2️⃣  Creating npm package structure..."
mkdir -p "$NPM_PACKAGE_DIR/binaries"

# Copy test binary to binaries folder with platform-specific name
PLATFORM=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map arch to Node.js naming
if [ "$ARCH" = "x86_64" ]; then
  ARCH="x64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
  ARCH="arm64"
fi

# Map platform to Node.js naming
if [ "$PLATFORM" = "darwin" ]; then
  BINARY_NAME="darwin-$ARCH"
elif [ "$PLATFORM" = "linux" ]; then
  BINARY_NAME="linux-$ARCH"
else
  echo -e "${YELLOW}⚠️  Unsupported platform: $PLATFORM${NC}"
  exit 1
fi

cp "$TEST_DIR/dtoforge-test" "$NPM_PACKAGE_DIR/binaries/$BINARY_NAME"
chmod +x "$NPM_PACKAGE_DIR/binaries/$BINARY_NAME"

# Copy npm package files
cp "$SCRIPT_DIR/install.js" "$NPM_PACKAGE_DIR/"
cp "$SCRIPT_DIR/bin-wrapper.js" "$NPM_PACKAGE_DIR/"
cp "$PROJECT_ROOT/README.md" "$NPM_PACKAGE_DIR/"

# Create package.json
cat > "$NPM_PACKAGE_DIR/package.json" <<EOF
{
  "name": "dtoforge-test",
  "version": "0.0.0-test",
  "description": "Test package for DtoForge",
  "bin": {
    "dtoforge": "./bin-wrapper.js"
  },
  "scripts": {
    "postinstall": "node install.js"
  },
  "files": [
    "README.md",
    "install.js",
    "bin-wrapper.js",
    "bin/",
    "binaries/"
  ]
}
EOF

echo -e "${GREEN}✅ Package structure created${NC}"

# Step 3: Create test project and install package
echo ""
echo "3️⃣  Creating test project and installing package..."
mkdir -p "$TEST_PROJECT_DIR"
cd "$TEST_PROJECT_DIR"

# Initialize npm project
npm init -y > /dev/null 2>&1

# Install the local package
npm install "$NPM_PACKAGE_DIR" > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo -e "${GREEN}✅ Package installed successfully${NC}"
else
  echo -e "${RED}❌ Failed to install package${NC}"
  exit 1
fi

# Step 4: Test npx command
echo ""
echo "4️⃣  Testing 'npx dtoforge --help'..."
npx dtoforge --help > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo -e "${GREEN}✅ npx command works${NC}"
else
  echo -e "${RED}❌ npx command failed${NC}"
  exit 1
fi

# Step 5: Test direct execution from node_modules/.bin
echo ""
echo "5️⃣  Testing direct execution from node_modules/.bin/dtoforge..."
./node_modules/.bin/dtoforge --help > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo -e "${GREEN}✅ Direct execution works${NC}"
else
  echo -e "${RED}❌ Direct execution failed${NC}"
  exit 1
fi

# Step 6: Verify binary was installed correctly
echo ""
echo "6️⃣  Verifying binary installation..."
if [ -f "./node_modules/dtoforge-test/bin/dtoforge-binary" ] || [ -f "./node_modules/dtoforge-test/bin/dtoforge.exe" ]; then
  echo -e "${GREEN}✅ Binary installed correctly${NC}"
else
  echo -e "${RED}❌ Binary not found in expected location${NC}"
  ls -la "./node_modules/dtoforge-test/bin/" || echo "bin directory not found"
  exit 1
fi

echo ""
echo "================================================"
echo -e "${GREEN}✅ All tests passed!${NC}"
echo "================================================"
echo ""
echo "📦 Package structure:"
echo "   - bin-wrapper.js: Entry point for npm"
echo "   - install.js: Copies platform-specific binary"
echo "   - binaries/: Platform-specific binaries"
echo ""
echo "🎯 Users can run: npx dtoforge --help"
echo "🌍 Or install globally: npm install -g dtoforge"
echo ""
