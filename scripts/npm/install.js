#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

function getPlatformBinary() {
  const platform = process.platform;
  const arch = process.arch;

  // Map Node.js platform/arch to our binary naming
  const platformMap = {
    'darwin': { x64: 'darwin-x64', arm64: 'darwin-arm64' },
    'linux': { x64: 'linux-x64', arm64: 'linux-arm64' },
    'win32': { x64: 'win32-x64.exe', arm64: 'win32-arm64.exe' }
  };

  if (!platformMap[platform] || !platformMap[platform][arch]) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  return platformMap[platform][arch];
}

function installBinary() {
  try {
    const binaryName = getPlatformBinary();
    const sourcePath = path.join(__dirname, 'binaries', binaryName);

    console.log(`Installing DtoForge for ${process.platform}-${process.arch}...`);

    if (!fs.existsSync(sourcePath)) {
      throw new Error(`Binary for ${process.platform}-${process.arch} not found`);
    }

    // Create bin directory
    const binDir = path.join(__dirname, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    // Copy the correct binary to bin/dtoForge
    const targetName = process.platform === 'win32' ? 'dtoForge.exe' : 'dtoForge';
    const targetPath = path.join(binDir, targetName);

    fs.copyFileSync(sourcePath, targetPath);

    // Make executable on Unix-like systems
    if (process.platform !== 'win32') {
      fs.chmodSync(targetPath, 0o755);
    }

    console.log(`✅ DtoForge installed successfully!`);
    console.log(`🎯 Use: npx dtoforge --help`);
    console.log(`🌍 Or install globally: npm install -g dtoforge`);

  } catch (error) {
    console.error(`❌ Installation failed: ${error.message}`);
    console.log(`
📥 Manual Installation:
Download the binary directly from:
https://github.com/eliraz-refael/dtoForge/releases/latest
`);
    process.exit(1);
  }
}

// Only run if this script is executed directly (not required)
if (require.main === module) {
  installBinary();
}

module.exports = { installBinary };
